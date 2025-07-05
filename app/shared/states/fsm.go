package states

import (
	"sync"
)

type UserState struct {
	CurrentAction string
	TemplateData  map[string]interface{}
}

var (
	userStates = make(map[int64]*UserState)
	mutex      sync.RWMutex
)

func SetUserState(userID int64, state *UserState) {
	mutex.Lock()
	defer mutex.Unlock()
	userStates[userID] = state
}

func GetUserState(userID int64) *UserState {
	mutex.RLock()
	defer mutex.RUnlock()
	return userStates[userID]
}

func ClearUserState(userID int64) {
	mutex.Lock()
	defer mutex.Unlock()
	delete(userStates, userID)
}
