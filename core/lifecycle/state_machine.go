package lifecycle

import "fmt"

var States = []string{"draft", "planned", "implementing", "reviewing", "validated", "archived"}

var transitions = map[string]map[string]bool{
	"draft":        {"planned": true},
	"planned":      {"implementing": true},
	"implementing": {"planned": true, "reviewing": true},
	"reviewing":    {"planned": true, "implementing": true, "validated": true},
	"validated":    {"reviewing": true},
	"archived":     {},
}

func CanTransition(from, to string) bool {
	return isState(from) && isState(to) && transitions[from][to]
}

func GateForTransition(_, to string) string {
	switch to {
	case "planned", "implementing":
		return "plan"
	case "reviewing":
		return "implement"
	case "validated":
		return "review"
	default:
		return ""
	}
}

func AssertTransition(from, to string) error {
	if to == "archived" {
		return fmt.Errorf("Use the archive command instead of transitioning directly to archived")
	}
	if !isState(from) {
		return fmt.Errorf("Unknown current state: %s", from)
	}
	if !isState(to) {
		return fmt.Errorf("Unknown target state: %s", to)
	}
	if !CanTransition(from, to) {
		return fmt.Errorf("Invalid state transition: %s -> %s", from, to)
	}
	return nil
}

func isState(value string) bool {
	_, exists := transitions[value]
	return exists
}
