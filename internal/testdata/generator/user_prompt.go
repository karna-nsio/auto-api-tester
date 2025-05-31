package generator

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
)

// UserPromptHandler handles interactive user prompts
type UserPromptHandler struct {
	prompts   []Prompt
	responses map[string]interface{}
	reader    *bufio.Reader
}

// Prompt represents a user prompt
type Prompt struct {
	Type     string                 `json:"type"`
	Question string                 `json:"question"`
	Context  map[string]interface{} `json:"context"`
	Options  []string               `json:"options"`
}

// NewUserPromptHandler creates a new user prompt handler
func NewUserPromptHandler() *UserPromptHandler {
	return &UserPromptHandler{
		prompts:   make([]Prompt, 0),
		responses: make(map[string]interface{}),
		reader:    bufio.NewReader(os.Stdin),
	}
}

// ConfirmMapping prompts the user to confirm or modify a schema mapping
func (h *UserPromptHandler) ConfirmMapping(ctx context.Context, mapping SchemaMapping) (SchemaMapping, error) {
	// Create prompt for mapping confirmation
	prompt := Prompt{
		Type: "mapping",
		Question: fmt.Sprintf("Please confirm or modify the mapping for table %s:\n\n"+
			"Current Mapping:\n"+
			"API Entity: %s\n"+
			"Field Mappings: %v\n"+
			"Business Rules: %v\n"+
			"Relationships: %v\n\n"+
			"Options:\n"+
			"1. Confirm (c)\n"+
			"2. Modify (m)\n"+
			"3. Skip (s)\n"+
			"Enter your choice: ", mapping.TableName, mapping.ApiEntityName, mapping.FieldMappings, mapping.BusinessRules, mapping.Relationships),
		Context: map[string]interface{}{
			"current_mapping": mapping,
		},
		Options: []string{"c", "m", "s"},
	}

	// Get user response
	response, err := h.getUserResponse(ctx, prompt)
	if err != nil {
		return mapping, fmt.Errorf("failed to get user response: %v", err)
	}

	// Handle user response
	switch response {
	case "c":
		return mapping, nil
	case "m":
		return h.handleMappingModification(ctx, mapping)
	case "s":
		return mapping, nil
	default:
		return mapping, fmt.Errorf("invalid response: %s", response)
	}
}

// ConfirmBusinessRule prompts the user to confirm a business rule
func (h *UserPromptHandler) ConfirmBusinessRule(ctx context.Context, rule BusinessRule) (BusinessRule, error) {
	prompt := Prompt{
		Type: "business_rule",
		Question: fmt.Sprintf("Please confirm the following business rule:\n\n"+
			"Type: %s\n"+
			"Condition: %s\n"+
			"Action: %s\n"+
			"Priority: %d\n\n"+
			"Options:\n"+
			"1. Confirm (c)\n"+
			"2. Modify (m)\n"+
			"3. Reject (r)\n"+
			"Enter your choice: ", rule.Type, rule.Condition, rule.Action, rule.Priority),
		Context: map[string]interface{}{
			"rule": rule,
		},
		Options: []string{"c", "m", "r"},
	}

	response, err := h.getUserResponse(ctx, prompt)
	if err != nil {
		return rule, fmt.Errorf("failed to get user response: %v", err)
	}

	switch response {
	case "c":
		return rule, nil
	case "m":
		return h.handleRuleModification(ctx, rule)
	case "r":
		return BusinessRule{}, nil
	default:
		return rule, fmt.Errorf("invalid response: %s", response)
	}
}

// getUserResponse gets a response from the user
func (h *UserPromptHandler) getUserResponse(ctx context.Context, prompt Prompt) (string, error) {
	fmt.Print(prompt.Question)

	response, err := h.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read user input: %v", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if !contains(prompt.Options, response) {
		return "", fmt.Errorf("invalid option: %s", response)
	}

	return response, nil
}

// handleMappingModification handles mapping modification by the user
func (h *UserPromptHandler) handleMappingModification(ctx context.Context, mapping SchemaMapping) (SchemaMapping, error) {
	fmt.Printf("\nModifying mapping for table %s\n", mapping.TableName)

	// Modify API entity name
	fmt.Printf("Current API entity name: %s\n", mapping.ApiEntityName)
	fmt.Print("Enter new API entity name (press Enter to keep current): ")
	newName, err := h.reader.ReadString('\n')
	if err != nil {
		return mapping, fmt.Errorf("failed to read API entity name: %v", err)
	}
	newName = strings.TrimSpace(newName)
	if newName != "" {
		mapping.ApiEntityName = newName
	}

	// Modify field mappings
	fmt.Println("\nCurrent field mappings:")
	for col, field := range mapping.FieldMappings {
		fmt.Printf("%s -> %s\n", col, field)
		fmt.Printf("Enter new field name for %s (press Enter to keep current): ", col)
		newField, err := h.reader.ReadString('\n')
		if err != nil {
			return mapping, fmt.Errorf("failed to read field mapping: %v", err)
		}
		newField = strings.TrimSpace(newField)
		if newField != "" {
			mapping.FieldMappings[col] = newField
		}
	}

	return mapping, nil
}

// handleRuleModification handles business rule modification by the user
func (h *UserPromptHandler) handleRuleModification(ctx context.Context, rule BusinessRule) (BusinessRule, error) {
	fmt.Printf("\nModifying business rule:\n"+
		"Type: %s\n"+
		"Condition: %s\n"+
		"Action: %s\n"+
		"Priority: %d\n", rule.Type, rule.Condition, rule.Action, rule.Priority)

	// Modify rule type
	fmt.Print("Enter new rule type (press Enter to keep current): ")
	newType, err := h.reader.ReadString('\n')
	if err != nil {
		return rule, fmt.Errorf("failed to read rule type: %v", err)
	}
	newType = strings.TrimSpace(newType)
	if newType != "" {
		rule.Type = newType
	}

	// Modify condition
	fmt.Print("Enter new condition (press Enter to keep current): ")
	newCondition, err := h.reader.ReadString('\n')
	if err != nil {
		return rule, fmt.Errorf("failed to read condition: %v", err)
	}
	newCondition = strings.TrimSpace(newCondition)
	if newCondition != "" {
		rule.Condition = newCondition
	}

	// Modify action
	fmt.Print("Enter new action (press Enter to keep current): ")
	newAction, err := h.reader.ReadString('\n')
	if err != nil {
		return rule, fmt.Errorf("failed to read action: %v", err)
	}
	newAction = strings.TrimSpace(newAction)
	if newAction != "" {
		rule.Action = newAction
	}

	return rule, nil
}

// contains checks if a string is in a slice
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
