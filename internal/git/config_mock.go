// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package git

import (
	"sync"
)

// Ensure, that GitConfigAPIMock does implement GitConfigAPI.
// If this is not the case, regenerate this file with moq.
var _ GitConfigAPI = &GitConfigAPIMock{}

// GitConfigAPIMock is a mock implementation of GitConfigAPI.
//
// 	func TestSomethingThatUsesGitConfigAPI(t *testing.T) {
//
// 		// make and configure a mocked GitConfigAPI
// 		mockedGitConfigAPI := &GitConfigAPIMock{
// 			CommitterDetailsFunc: func() (string, string, error) {
// 				panic("mock out the CommitterDetails method")
// 			},
// 		}
//
// 		// use mockedGitConfigAPI in code that requires GitConfigAPI
// 		// and then make assertions.
//
// 	}
type GitConfigAPIMock struct {
	// CommitterDetailsFunc mocks the CommitterDetails method.
	CommitterDetailsFunc func() (string, string, error)

	// calls tracks calls to the methods.
	calls struct {
		// CommitterDetails holds details about calls to the CommitterDetails method.
		CommitterDetails []struct {
		}
	}
	lockCommitterDetails sync.RWMutex
}

// CommitterDetails calls CommitterDetailsFunc.
func (mock *GitConfigAPIMock) CommitterDetails() (string, string, error) {
	if mock.CommitterDetailsFunc == nil {
		panic("GitConfigAPIMock.CommitterDetailsFunc: method is nil but GitConfigAPI.CommitterDetails was just called")
	}
	callInfo := struct {
	}{}
	mock.lockCommitterDetails.Lock()
	mock.calls.CommitterDetails = append(mock.calls.CommitterDetails, callInfo)
	mock.lockCommitterDetails.Unlock()
	return mock.CommitterDetailsFunc()
}

// CommitterDetailsCalls gets all the calls that were made to CommitterDetails.
// Check the length with:
//     len(mockedGitConfigAPI.CommitterDetailsCalls())
func (mock *GitConfigAPIMock) CommitterDetailsCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockCommitterDetails.RLock()
	calls = mock.calls.CommitterDetails
	mock.lockCommitterDetails.RUnlock()
	return calls
}
