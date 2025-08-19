package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"states-shuffler/types"
)

func main() {
	const inputJSON = "json/states.json"
	const outputTxt = "modded/modded_00_west_europe.txt"

	rand.Seed(time.Now().UnixNano())

	jsonData, err := os.ReadFile(inputJSON)
	if err != nil {
		fmt.Printf("Ошибка чтения JSON %s: %v\n", inputJSON, err)
		return
	}

	var states []types.State
	if err := json.Unmarshal(jsonData, &states); err != nil {
		fmt.Printf("Ошибка десериализации JSON: %v\n", err)
		return
	}

	// Модификация ресурсов (фиксированные параметры)
	for i := range states {
		modifyResources(&states[i], 50, "bg_oil_extraction", 75)
	}

	outputTxtContent := generateTxtFromStates(states)
	if err := os.MkdirAll("modded", 0755); err != nil {
		fmt.Printf("Ошибка создания папки modded: %v\n", err)
		return
	}

	if err := os.WriteFile(outputTxt, []byte(outputTxtContent), 0644); err != nil {
		fmt.Printf("Ошибка записи TXT в %s: %v\n", outputTxt, err)
		return
	}

	fmt.Printf("Конвертация завершена! Файл %s создан.\n", outputTxt)
}

func modifyResources(state *types.State, addRandom int, newResource string, newResourceValue int) {
	// Добавление случайных значений к существующим ресурсам
	for res := range state.CappedResources {
		state.CappedResources[res] += rand.Intn(addRandom) + 1
	}

	// Добавление нового ресурса
	if newResource != "" {
		if newResourceValue == 0 {
			newResourceValue = rand.Intn(91) + 10 // 10-100
		}
		state.CappedResources[newResource] = newResourceValue
	}
}

func generateTxtFromStates(states []types.State) string {
	var sb strings.Builder

	for _, state := range states {
		sb.WriteString(fmt.Sprintf("%s = {\n", state.Name))
		sb.WriteString(fmt.Sprintf("    id = %d\n", state.ID))
		sb.WriteString(fmt.Sprintf("    subsistence_building = %q\n", state.SubsistenceBuilding))

		sb.WriteString("    provinces = { ")
		sb.WriteString(strings.Join(quoteSlice(state.Provinces), " "))
		sb.WriteString(" }\n")

		if len(state.Impassable) > 0 {
			sb.WriteString("    impassable = { ")
			sb.WriteString(strings.Join(quoteSlice(state.Impassable), " "))
			sb.WriteString(" }\n")
		}

		if len(state.PrimeLand) > 0 {
			sb.WriteString("    prime_land = { ")
			sb.WriteString(strings.Join(quoteSlice(state.PrimeLand), " "))
			sb.WriteString(" }\n")
		}

		sb.WriteString("    traits = { ")
		sb.WriteString(strings.Join(quoteSlice(state.Traits), " "))
		sb.WriteString(" }\n")

		sb.WriteString(fmt.Sprintf("    city = %q\n", state.City))
		if state.Port != "" {
			sb.WriteString(fmt.Sprintf("    port = %q\n", state.Port))
		}
		sb.WriteString(fmt.Sprintf("    farm = %q\n", state.Farm))
		if state.Mine != "" {
			sb.WriteString(fmt.Sprintf("    mine = %q\n", state.Mine))
		}
		if state.Wood != "" {
			sb.WriteString(fmt.Sprintf("    wood = %q\n", state.Wood))
		}

		sb.WriteString(fmt.Sprintf("    arable_land = %d\n", state.ArableLand))

		sb.WriteString("    arable_resources = { ")
		sb.WriteString(strings.Join(quoteSlice(state.ArableResources), " "))
		sb.WriteString(" }\n")

		sb.WriteString("    capped_resources = {\n")
		for res, val := range state.CappedResources {
			sb.WriteString(fmt.Sprintf("        %s = %d\n", res, val))
		}
		sb.WriteString("    }\n")

		if state.Resource != nil {
			sb.WriteString("    resource = {\n")
			sb.WriteString(fmt.Sprintf("        type = %q\n", state.Resource.Type))
			sb.WriteString(fmt.Sprintf("        undiscovered_amount = %d\n", state.Resource.UndiscoveredAmount))
			sb.WriteString("    }\n")
		}

		if state.NavalExitID != 0 {
			sb.WriteString(fmt.Sprintf("    naval_exit_id = %d\n", state.NavalExitID))
		}

		sb.WriteString("}\n\n")
	}

	return sb.String()
}

func quoteSlice(items []string) []string {
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = fmt.Sprintf("%q", item)
	}
	return quoted
}