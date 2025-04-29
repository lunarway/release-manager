package kubernetes

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestAnnotateStatefulSet(t *testing.T) {
	t.Run("successfully annotates statefulset", func(t *testing.T) {
		// Setup
		ss := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-statefulset",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					artifactIDAnnotationKey: "test-artifact",
				},
			},
		}

		clientset := fake.NewSimpleClientset(ss)

		// Test
		err := annotateStatefulSet(context.Background(), clientset, ss)

		// Assert
		assert.NoError(t, err)

		// Verify the StatefulSet was updated with observed annotation
		updated, err := clientset.AppsV1().StatefulSets(ss.Namespace).Get(context.Background(), ss.Name, metav1.GetOptions{})
		assert.NoError(t, err)
		assert.Equal(t, "test-artifact", updated.Annotations[observedAnnotationKey])
	})

	t.Run("handles nil annotations map", func(t *testing.T) {
		// Setup
		ss := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-statefulset",
				Namespace: "test-namespace",
			},
		}

		clientset := fake.NewSimpleClientset(ss)

		// Test
		err := annotateStatefulSet(context.Background(), clientset, ss)

		// Assert
		assert.NoError(t, err)

		// Verify annotations were created and set
		updated, err := clientset.AppsV1().StatefulSets(ss.Namespace).Get(context.Background(), ss.Name, metav1.GetOptions{})
		assert.NoError(t, err)
		assert.NotNil(t, updated.Annotations)
	})

	t.Run("handles non-existent statefulset", func(t *testing.T) {
		// Setup
		ss := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "non-existent",
				Namespace: "test-namespace",
			},
		}

		clientset := fake.NewSimpleClientset()

		// Test
		err := annotateStatefulSet(context.Background(), clientset, ss)

		// Assert
		assert.Error(t, err)
	})

	t.Run("handles concurrent modification", func(t *testing.T) {
		// Setup initial StatefulSet
		ss := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-statefulset",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					artifactIDAnnotationKey: "test-artifact",
				},
			},
		}

		clientset := fake.NewSimpleClientset(ss)

		// Simulate concurrent modification by having another process modify the StatefulSet
		// between our Get and Update calls
		count := 0
		clientset.PrependReactor("get", "statefulsets", func(action ktesting.Action) (bool, runtime.Object, error) {
			if count == 0 {
				count++
				return false, nil, nil // let the first Get go through normally
			}
			// On second Get (after first Update fails), return StatefulSet with updated annotations
			modifiedSS := ss.DeepCopy()
			if modifiedSS.Annotations == nil {
				modifiedSS.Annotations = make(map[string]string)
			}
			modifiedSS.Annotations[artifactIDAnnotationKey] = "test-artifact"
			modifiedSS.Annotations[observedAnnotationKey] = "test-artifact"
			return true, modifiedSS, nil
		})

		// First update fails with conflict
		firstUpdate := true
		clientset.PrependReactor("update", "statefulsets", func(action ktesting.Action) (bool, runtime.Object, error) {
			if firstUpdate {
				firstUpdate = false
				return true, nil, errors.New("the object has been modified; please apply your changes to the latest version and try again")
			}
			return false, nil, nil // let subsequent updates go through
		})

		// Test
		err := annotateStatefulSet(context.Background(), clientset, ss)

		// Assert
		assert.NoError(t, err)

		// Verify the StatefulSet was updated with observed annotation
		updated, err := clientset.AppsV1().StatefulSets(ss.Namespace).Get(context.Background(), ss.Name, metav1.GetOptions{})
		assert.NoError(t, err)
		assert.Equal(t, "test-artifact", updated.Annotations[observedAnnotationKey])
	})
}
