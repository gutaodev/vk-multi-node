package node

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gutaodev/vk-multi-node/log"
	"github.com/gutaodev/vk-multi-node/trace"
	pkgerrors "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	coordclientset "k8s.io/client-go/kubernetes/typed/coordination/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/clock"
)

type MultiNodeProvider interface { // nolint:golint
	Ping(context.Context) error

	NotifyNodeStatus(ctx context.Context, cb func(*corev1.Node))

	RetrieveNodes(context.Context) ([]corev1.Node, error)
}

func NewMultiNodeController(p MultiNodeProvider, node *corev1.Node, nodes v1.NodeInterface, opts ...MultiNodeControllerOpt) (*MultiNodeController, error) {
	nodeList, err := p.RetrieveNodes(context.Background())
	if err != nil {
		return nil, err
	}
	sns := make([]corev1.Node, len(nodeList))
	for i := range nodeList {
		sns[i] = *node.DeepCopy()
	}
	n := &MultiNodeController{
		p:           p,
		serverNodes: sns,
		nodes:       nodes,
		chReady:     make(chan struct{}),
		chDone:      make(chan struct{}),
	}
	for _, o := range opts {
		if err := o(n); err != nil {
			return nil, pkgerrors.Wrap(err, "error applying node option")
		}
	}

	if n.pingInterval == time.Duration(0) {
		n.pingInterval = DefaultPingInterval
	}
	if n.statusInterval == time.Duration(0) {
		n.statusInterval = DefaultStatusUpdateInterval
	}

	n.nodePingController = newNodePingController(n.p, n.pingInterval, n.pingTimeout)

	return n, nil
}

// NodeControllerOpt are the functional options used for configuring a node
type MultiNodeControllerOpt func(*MultiNodeController) error // nolint:golint

func WithMultiNodeEnableLeaseV1(client coordclientset.LeaseInterface, leaseDurationSeconds int32) NodeControllerOpt {
	if leaseDurationSeconds == 0 {
		leaseDurationSeconds = DefaultLeaseDuration
	}

	interval := float64(leaseDurationSeconds) * DefaultRenewIntervalFraction
	intervalDuration := time.Second * time.Duration(int(interval))

	return WithNodeEnableLeaseV1WithRenewInterval(client, leaseDurationSeconds, intervalDuration)
}

func WithMultiNodeEnableLeaseV1WithRenewInterval(client coordclientset.LeaseInterface, leaseDurationSeconds int32, interval time.Duration) MultiNodeControllerOpt {
	if client == nil {
		panic("client is nil")
	}

	if leaseDurationSeconds == 0 {
		leaseDurationSeconds = DefaultLeaseDuration
	}

	return func(n *MultiNodeController) error {
		if n.leaseController != nil {
			return ErrConflictingLeaseControllerConfiguration
		}

		leaseController, err := newMultiLeaseControllerWithRenewInterval(
			&clock.RealClock{},
			client,
			leaseDurationSeconds,
			interval,
			n,
		)
		if err != nil {
			return fmt.Errorf("Unable to configure lease controller: %w", err)
		}

		n.leaseController = leaseController
		return nil
	}
}

func WithMultiNodePingTimeout(timeout time.Duration) MultiNodeControllerOpt {
	return func(n *MultiNodeController) error {
		n.pingTimeout = &timeout
		return nil
	}
}

func WithMultiNodePingInterval(d time.Duration) MultiNodeControllerOpt {
	return func(n *MultiNodeController) error {
		n.pingInterval = d
		return nil
	}
}

func WithMultiNodeStatusUpdateInterval(d time.Duration) MultiNodeControllerOpt {
	return func(n *MultiNodeController) error {
		n.statusInterval = d
		return nil
	}
}

func WithMultiNodeStatusUpdateErrorHandler(h ErrorHandler) MultiNodeControllerOpt {
	return func(n *MultiNodeController) error {
		n.nodeStatusUpdateErrorHandler = h
		return nil
	}
}

type MultiNodeController struct { // nolint:golint
	p MultiNodeProvider

	// serverNode must be updated each time it is updated in API Server
	serverNodeLock sync.Mutex
	serverNodes    []corev1.Node
	nodes          v1.NodeInterface

	leaseController *multiLeaseController

	pingInterval   time.Duration
	statusInterval time.Duration
	chStatusUpdate chan *corev1.Node

	nodeStatusUpdateErrorHandler ErrorHandler

	// chReady is closed once the controller is ready to start the control loop
	chReady chan struct{}
	// chDone is closed once the control loop has exited
	chDone chan struct{}
	errMu  sync.Mutex
	err    error

	nodePingController *nodePingController
	pingTimeout        *time.Duration

	group wait.Group
}

func (n *MultiNodeController) Run(ctx context.Context) (retErr error) {
	defer func() {
		n.errMu.Lock()
		n.err = retErr
		n.errMu.Unlock()
		close(n.chDone)
	}()

	n.chStatusUpdate = make(chan *corev1.Node, 1)
	n.p.NotifyNodeStatus(ctx, func(node *corev1.Node) {
		n.chStatusUpdate <- node
	})

	n.group.StartWithContext(ctx, n.nodePingController.Run)

	nodeList, err := n.p.RetrieveNodes(ctx)
	if err != nil {
		return err
	}

	var providerNodes []corev1.Node
	for _, node := range nodeList {
		n.serverNodeLock.Lock()
		providerNodes = append(providerNodes, *node.DeepCopy())
		n.serverNodeLock.Unlock()
	}

	if err := n.ensureNode(ctx, providerNodes); err != nil {
		return err
	}

	if n.leaseController != nil {
		log.G(ctx).WithField("leaseController", n.leaseController).Debug("Starting leasecontroller")
		n.group.StartWithContext(ctx, n.leaseController.Run)
	}

	return n.controlLoop(ctx, providerNodes)
}

func (n *MultiNodeController) Done() <-chan struct{} {
	return n.chDone
}

func (n *MultiNodeController) Err() error {
	n.errMu.Lock()
	defer n.errMu.Unlock()
	return n.err
}

func (n *MultiNodeController) ensureNode(ctx context.Context, providerNodes []corev1.Node) (err error) {
	ctx, span := trace.StartSpan(ctx, "node.ensureNode")
	defer span.End()
	defer func() {
		span.SetStatus(err)
	}()

	for i, e := range providerNodes {
		err = n.updateStatus(ctx, &e, true)
		if err == nil || !errors.IsNotFound(err) {
			return err
		}
		n.serverNodeLock.Lock()
		serverNode := &e
		n.serverNodeLock.Unlock()
		node, err := n.nodes.Create(ctx, serverNode, metav1.CreateOptions{})
		if err != nil {
			return pkgerrors.Wrap(err, "error registering node with kubernetes")
		}

		for j, o := range n.serverNodes {
			if o.Name == e.Name {
				n.serverNodeLock.Lock()
				n.serverNodes[j] = *node
				n.serverNodeLock.Unlock()
			}
		}

		// Bad things will happen if the node is deleted in k8s and recreated by someone else
		// we rely on this persisting
		providerNodes[i].ObjectMeta.Name = node.Name
		providerNodes[i].ObjectMeta.Namespace = node.Namespace
		providerNodes[i].ObjectMeta.UID = node.UID
	}

	return nil
}

func (n *MultiNodeController) Ready() <-chan struct{} {
	return n.chReady
}

func (n *MultiNodeController) controlLoop(ctx context.Context, providerNodes []corev1.Node) error {
	defer n.group.Wait()

	var sleepInterval time.Duration
	if n.leaseController == nil {
		log.G(ctx).WithField("pingInterval", n.pingInterval).Debug("lease controller is not enabled, updating node status in Kube API server at Ping Time Interval")
		sleepInterval = n.pingInterval
	} else {
		log.G(ctx).WithField("statusInterval", n.statusInterval).Debug("lease controller in use, updating at statusInterval")
		sleepInterval = n.statusInterval
	}

	loop := func() bool {
		ctx, span := trace.StartSpan(ctx, "node.controlLoop.loop")
		defer span.End()

		var timer *time.Timer
		ctx = span.WithField(ctx, "sleepTime", n.pingInterval)
		timer = time.NewTimer(sleepInterval)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return true
		case updated := <-n.chStatusUpdate:
			log.G(ctx).Debug("Received node status update")

			var pn corev1.Node
			for _, e := range providerNodes {
				if updated.Name == e.Name {
					pn = e
					break
				}
			}
			pn.Status = updated.Status
			pn.ObjectMeta.Annotations = updated.Annotations
			pn.ObjectMeta.Labels = updated.Labels
			if err := n.updateStatus(ctx, &pn, false); err != nil {
				log.G(ctx).WithError(err).Error("Error handling node status update")
			}
		case <-timer.C:
			for _, e := range providerNodes {
				if err := n.updateStatus(ctx, &e, false); err != nil {
					log.G(ctx).WithError(err).Error("Error handling node status update")
				}
			}

		}
		return false
	}

	close(n.chReady)
	for {
		shouldTerminate := loop()
		if shouldTerminate {
			return nil
		}
	}
}

func (n *MultiNodeController) updateStatus(ctx context.Context, providerNode *corev1.Node, skipErrorCb bool) (err error) {
	ctx, span := trace.StartSpan(ctx, "node.updateStatus")
	defer span.End()
	defer func() {
		span.SetStatus(err)
	}()

	if result, err := n.nodePingController.getResult(ctx); err != nil {
		return err
	} else if result.error != nil {
		return fmt.Errorf("Not updating node status because node ping failed: %w", result.error)
	}

	updateNodeStatusHeartbeat(providerNode)

	node, err := updateNodeStatus(ctx, n.nodes, providerNode)
	if err != nil {
		if skipErrorCb || n.nodeStatusUpdateErrorHandler == nil {
			return err
		}
		if err := n.nodeStatusUpdateErrorHandler(ctx, err); err != nil {
			return err
		}

		// This might have recreated the node, which may cause problems with our leases until a node update succeeds
		node, err = updateNodeStatus(ctx, n.nodes, providerNode)
		if err != nil {
			return err
		}
	}

	for i, e := range n.serverNodes {
		if e.Name == node.Name {
			n.serverNodeLock.Lock()
			n.serverNodes[i] = *node
			n.serverNodeLock.Unlock()
			break
		}
	}

	return nil
}

// Returns a copy of the server node object
func (n *MultiNodeController) getServerNode(ctx context.Context) (*corev1.Node, error) {
	n.serverNodeLock.Lock()
	defer n.serverNodeLock.Unlock()
	if n.serverNodes == nil {
		return nil, pkgerrors.New("Server node does not yet exist")
	}
	// todo(gutao)
	return n.serverNodes[0].DeepCopy(), nil
}

// todo(@gutao)
func prepareThreewayPatchBytesForNodeStatus_todo(nodeFromProvider, apiServerNode *corev1.Node) ([]byte, error) {
	// We use these two values to calculate a patch. We use a three-way patch in order to avoid
	// causing state regression server side. For example, let's consider the scenario:
	/*
		UML Source:
		@startuml
		participant VK
		participant K8s
		participant ExternalUpdater
		note right of VK: Updates internal node conditions to [A, B]
		VK->K8s: Patch Upsert [A, B]
		note left of K8s: Node conditions are [A, B]
		ExternalUpdater->K8s: Patch Upsert [C]
		note left of K8s: Node Conditions are [A, B, C]
		note right of VK: Updates internal node conditions to [A]
		VK->K8s: Patch: delete B, upsert A\nThis is where things go wrong,\nbecause the patch is written to replace all node conditions\nit overwrites (drops) [C]
		note left of K8s: Node Conditions are [A]\nNode condition C from ExternalUpdater is no longer present
		@enduml
			     ,--.                                                        ,---.          ,---------------.
		     |VK|                                                        |K8s|          |ExternalUpdater|
		     `+-'                                                        `-+-'          `-------+-------'
		      |  ,------------------------------------------!.             |                    |
		      |  |Updates internal node conditions to [A, B]|_\            |                    |
		      |  `--------------------------------------------'            |                    |
		      |                     Patch Upsert [A, B]                    |                    |
		      | ----------------------------------------------------------->                    |
		      |                                                            |                    |
		      |                              ,--------------------------!. |                    |
		      |                              |Node conditions are [A, B]|_\|                    |
		      |                              `----------------------------'|                    |
		      |                                                            |  Patch Upsert [C]  |
		      |                                                            | <-------------------
		      |                                                            |                    |
		      |                           ,-----------------------------!. |                    |
		      |                           |Node Conditions are [A, B, C]|_\|                    |
		      |                           `-------------------------------'|                    |
		      |  ,---------------------------------------!.                |                    |
		      |  |Updates internal node conditions to [A]|_\               |                    |
		      |  `-----------------------------------------'               |                    |
		      | Patch: delete B, upsert A                                  |                    |
		      | This is where things go wrong,                             |                    |
		      | because the patch is written to replace all node conditions|                    |
		      | it overwrites (drops) [C]                                  |                    |
		      | ----------------------------------------------------------->                    |
		      |                                                            |                    |
		     ,----------------------------------------------------------!. |                    |
		     |Node Conditions are [A]                                   |_\|                    |
		     |Node condition C from ExternalUpdater is no longer present  ||                    |
		     `------------------------------------------------------------'+-.          ,-------+-------.
		     |VK|                                                        |K8s|          |ExternalUpdater|
		     `--'                                                        `---'          `---------------'
	*/
	// In order to calculate that last patch to delete B, and upsert C, we need to know that C was added by
	// "someone else". So, we keep track of our last applied value, and our current value. We then generate
	// our patch based on the diff of these and *not* server side state.
	oldVKStatus, ok1 := apiServerNode.Annotations[virtualKubeletLastNodeAppliedNodeStatus]
	oldVKObjectMeta, ok2 := apiServerNode.Annotations[virtualKubeletLastNodeAppliedObjectMeta]

	oldNode := corev1.Node{}
	// Check if there were no labels, which means someone else probably created the node, or this is an upgrade. Either way, we will consider
	// ourselves as never having written the node object before, so oldNode will be left empty. We will overwrite values if
	// our new node conditions / status / objectmeta have them
	if ok1 && ok2 {
		err := json.Unmarshal([]byte(oldVKObjectMeta), &oldNode.ObjectMeta)
		if err != nil {
			return nil, pkgerrors.Wrapf(err, "Cannot unmarshal old node object metadata (key: %q): %q", virtualKubeletLastNodeAppliedObjectMeta, oldVKObjectMeta)
		}
		err = json.Unmarshal([]byte(oldVKStatus), &oldNode.Status)
		if err != nil {
			return nil, pkgerrors.Wrapf(err, "Cannot unmarshal old node status (key: %q): %q", virtualKubeletLastNodeAppliedNodeStatus, oldVKStatus)
		}
	}

	// newNode is the representation of the node the provider "wants"
	newNode := corev1.Node{}
	newNode.ObjectMeta = simplestObjectMetadata(&apiServerNode.ObjectMeta, &nodeFromProvider.ObjectMeta)
	nodeFromProvider.Status.DeepCopyInto(&newNode.Status)

	// virtualKubeletLastNodeAppliedObjectMeta must always be set before virtualKubeletLastNodeAppliedNodeStatus,
	// otherwise we capture virtualKubeletLastNodeAppliedNodeStatus in virtualKubeletLastNodeAppliedObjectMeta,
	// which is wrong
	virtualKubeletLastNodeAppliedObjectMetaBytes, err := json.Marshal(newNode.ObjectMeta)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "Cannot marshal object meta from provider")
	}
	newNode.Annotations[virtualKubeletLastNodeAppliedObjectMeta] = string(virtualKubeletLastNodeAppliedObjectMetaBytes)

	virtualKubeletLastNodeAppliedNodeStatusBytes, err := json.Marshal(newNode.Status)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "Cannot marshal node status from provider")
	}
	newNode.Annotations[virtualKubeletLastNodeAppliedNodeStatus] = string(virtualKubeletLastNodeAppliedNodeStatusBytes)
	// Generate three way patch from oldNode -> newNode, without deleting the changes in api server
	// Should we use the Kubernetes serialization / deserialization libraries here?
	oldNodeBytes, err := json.Marshal(oldNode)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "Cannot marshal old node bytes")
	}
	newNodeBytes, err := json.Marshal(newNode)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "Cannot marshal new node bytes")
	}
	apiServerNodeBytes, err := json.Marshal(apiServerNode)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "Cannot marshal api server node")
	}
	schema, err := strategicpatch.NewPatchMetaFromStruct(&corev1.Node{})
	if err != nil {
		return nil, pkgerrors.Wrap(err, "Cannot get patch schema from node")
	}
	return strategicpatch.CreateThreeWayMergePatch(oldNodeBytes, newNodeBytes, apiServerNodeBytes, schema, true)
}

// todo, 更新节点状态，【实现】
func updateNodeStatus_todo(ctx context.Context, nodes v1.NodeInterface, nodeFromProvider *corev1.Node) (_ *corev1.Node, retErr error) {
	ctx, span := trace.StartSpan(ctx, "UpdateNodeStatus")
	defer func() {
		span.End()
		span.SetStatus(retErr)
	}()

	var updatedNode *corev1.Node
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		apiServerNode, err := nodes.Get(ctx, nodeFromProvider.Name, emptyGetOptions)
		if err != nil {
			return err
		}
		ctx = addNodeAttributes(ctx, span, apiServerNode)
		log.G(ctx).Debug("got node from api server")

		patchBytes, err := prepareThreewayPatchBytesForNodeStatus(nodeFromProvider, apiServerNode)
		if err != nil {
			return pkgerrors.Wrap(err, "Cannot generate patch")
		}
		log.G(ctx).WithError(err).WithField("patch", string(patchBytes)).Debug("Generated three way patch")

		updatedNode, err = nodes.Patch(ctx, nodeFromProvider.Name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{}, "status")
		if err != nil {
			// We cannot wrap this error because the kubernetes error module doesn't understand wrapping
			log.G(ctx).WithField("patch", string(patchBytes)).WithError(err).Warn("Failed to patch node status")
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	log.G(ctx).WithField("node.resourceVersion", updatedNode.ResourceVersion).
		WithField("node.Status.Conditions", updatedNode.Status.Conditions).
		Debug("updated node status in api server")
	return updatedNode, nil
}

func updateNodeStatusHeartbeat_todo(n *corev1.Node) {
	now := metav1.NewTime(time.Now())
	for i := range n.Status.Conditions {
		n.Status.Conditions[i].LastHeartbeatTime = now
	}
}

func addNodeAttributes_todo(ctx context.Context, span trace.Span, n *corev1.Node) context.Context {
	return span.WithFields(ctx, log.Fields{
		"node.UID":     string(n.UID),
		"node.name":    n.Name,
		"node.cluster": n.ClusterName,
		"node.taints":  taintsStringer(n.Spec.Taints),
	})
}

// Provides the simplest object metadata to match the previous object. Name, namespace, UID. It copies labels and
// annotations from the second object if defined. It exempts the patch metadata
func simplestObjectMetadata_todo(baseObjectMeta, objectMetaWithLabelsAndAnnotations *metav1.ObjectMeta) metav1.ObjectMeta {
	ret := metav1.ObjectMeta{
		Namespace:   baseObjectMeta.Namespace,
		Name:        baseObjectMeta.Name,
		UID:         baseObjectMeta.UID,
		Annotations: make(map[string]string),
	}
	if objectMetaWithLabelsAndAnnotations != nil {
		if objectMetaWithLabelsAndAnnotations.Labels != nil {
			ret.Labels = objectMetaWithLabelsAndAnnotations.Labels
		} else {
			ret.Labels = make(map[string]string)
		}
		if objectMetaWithLabelsAndAnnotations.Annotations != nil {
			// We want to copy over all annotations except the special embedded ones.
			for key := range objectMetaWithLabelsAndAnnotations.Annotations {
				if key == virtualKubeletLastNodeAppliedNodeStatus || key == virtualKubeletLastNodeAppliedObjectMeta {
					continue
				}
				ret.Annotations[key] = objectMetaWithLabelsAndAnnotations.Annotations[key]
			}
		}
	}
	return ret
}
