package workflow

import "fmt"

// ActionRegistry manages action implementations
type ActionRegistry struct {
	actions map[string]Action
}

// NewActionRegistry creates a new action registry
func NewActionRegistry() *ActionRegistry {
	return &ActionRegistry{
		actions: make(map[string]Action),
	}
}

// RegisterAction registers an action implementation
func (r *ActionRegistry) RegisterAction(actionType string, action Action) {
	r.actions[actionType] = action
}

// GetAction retrieves an action by type
func (r *ActionRegistry) GetAction(actionType string) (Action, error) {
	action, exists := r.actions[actionType]
	if !exists {
		return nil, fmt.Errorf("unknown action type: %s", actionType)
	}
	return action, nil
}
