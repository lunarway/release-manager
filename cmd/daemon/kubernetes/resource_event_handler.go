package kubernetes

import "k8s.io/client-go/tools/cache"

type ResourceEventHandlerFactory func(cache.ResourceEventHandlerFuncs) cache.ResourceEventHandler

// ResourceEventHandlerFuncs is a cache.ResourceEventHandler that can be
// configured to skip event handlers based on a ShouldProcess func.
type ResourceEventHandlerFuncs struct {
	ShouldProcess func() bool
	cache.ResourceEventHandlerFuncs
}

// OnAdd calls AddFunc if it's not nil.
func (r ResourceEventHandlerFuncs) OnAdd(obj interface{}) {
	if r.AddFunc != nil && r.ShouldProcess() {
		r.AddFunc(obj)
	}
}

// OnUpdate calls UpdateFunc if it's not nil.
func (r ResourceEventHandlerFuncs) OnUpdate(oldObj, newObj interface{}) {
	if r.UpdateFunc != nil && r.ShouldProcess() {
		r.UpdateFunc(oldObj, newObj)
	}
}

// OnDelete calls DeleteFunc if it's not nil.
func (r ResourceEventHandlerFuncs) OnDelete(obj interface{}) {
	if r.DeleteFunc != nil && r.ShouldProcess() {
		r.DeleteFunc(obj)
	}
}
