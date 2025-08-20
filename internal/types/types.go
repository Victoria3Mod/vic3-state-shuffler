package types

// State представляет регион из файла Victoria 3
type State struct {
	Name                string         `json:"name"`
	ID                  int            `json:"id"`
	SubsistenceBuilding string         `json:"subsistence_building"`
	Provinces           []string       `json:"provinces"`
	Impassable          []string       `json:"impassable,omitempty"`
	PrimeLand           []string       `json:"prime_land,omitempty"`
	Traits              []string       `json:"traits"`
	City                string         `json:"city"`
	Port                string         `json:"port,omitempty"`
	Farm                string         `json:"farm"`
	Mine                string         `json:"mine,omitempty"`
	Wood                string         `json:"wood,omitempty"`
	ArableLand          int            `json:"arable_land"`
	ArableResources     []string       `json:"arable_resources"`
	CappedResources     map[string]int `json:"capped_resources"`
	Resource            *Resource      `json:"resource,omitempty"`
	NavalExitID         int            `json:"naval_exit_id,omitempty"`
}

// Resource для вложенного блока resource
type Resource struct {
	Type               string `json:"type"`
	UndiscoveredAmount int    `json:"undiscovered_amount"`
}
