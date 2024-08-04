package node

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type MultiNodeProviderV1 struct {
	notify          func(*corev1.Node)
	updateReady     chan struct{}
	lowerNodeClient v1.NodeInterface
}

func (n *MultiNodeProviderV1) Ping(ctx context.Context) error {
	return ctx.Err()
}

func (n *MultiNodeProviderV1) NotifyNodeStatus(_ context.Context, f func(*corev1.Node)) {
	n.notify = f
	// This is a little sloppy and assumes `NotifyNodeStatus` is only called once, which is indeed currently true.
	// The reason a channel is preferred here is so we can use a context in `UpdateStatus` to cancel waiting for this.
	close(n.updateReady)
}

func (n *MultiNodeProviderV1) RetrieveNodes(ctx context.Context) ([]corev1.Node, error) {
	nodeList, err := n.lowerNodeClient.List(ctx, metav1.ListOptions{})
	if err != nil || nodeList == nil {
		return nil, err
	}
	return nodeList.Items, nil
}

// UpdateStatus sends a node status update to the node controller
func (n *MultiNodeProviderV1) UpdateStatus(ctx context.Context, node *corev1.Node) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-n.updateReady:
	}

	n.notify(node)
	return nil
}

func NewMultiNodeProviderV1(client v1.NodeInterface) *MultiNodeProviderV1 {
	return &MultiNodeProviderV1{
		updateReady:     make(chan struct{}),
		lowerNodeClient: client,
	}
}
