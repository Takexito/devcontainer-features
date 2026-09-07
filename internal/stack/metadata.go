package stack

import "encoding/json"

// FromMetadata достаёт маркер из метки devcontainer.metadata контейнера:
// CLI складывает туда customizations из devcontainer.json, так что стек
// известен без чтения файла проекта. Метка — массив записей метаданных,
// но на всякий случай принимаем и одиночный объект.
func FromMetadata(label string) (Marker, bool) {
	type entry struct {
		Customizations struct {
			Devc *Marker `json:"devc"`
		} `json:"customizations"`
	}
	var entries []entry
	if err := json.Unmarshal([]byte(label), &entries); err != nil {
		var one entry
		if err := json.Unmarshal([]byte(label), &one); err != nil {
			return Marker{}, false
		}
		entries = []entry{one}
	}
	for _, e := range entries {
		if e.Customizations.Devc != nil && e.Customizations.Devc.Stack != "" {
			return *e.Customizations.Devc, true
		}
	}
	return Marker{}, false
}
