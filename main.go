package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// State представляет регион из файла Victoria 3
type State struct {
	Name              string            `json:"name"`
	ID                int               `json:"id"`
	SubsistenceBuilding string          `json:"subsistence_building"`
	Provinces         []string          `json:"provinces"`
	Impassable        []string          `json:"impassable,omitempty"`
	PrimeLand         []string          `json:"prime_land,omitempty"`
	Traits            []string          `json:"traits"`
	City              string            `json:"city"`
	Port              string            `json:"port,omitempty"`
	Farm              string            `json:"farm"`
	Mine              string            `json:"mine,omitempty"`
	Wood              string            `json:"wood,omitempty"`
	ArableLand        int               `json:"arable_land"`
	ArableResources   []string          `json:"arable_resources"`
	CappedResources   map[string]int    `json:"capped_resources"`
	Resource          *Resource         `json:"resource,omitempty"`
	NavalExitID       int               `json:"naval_exit_id,omitempty"`
}

// Resource для вложенного блока resource (e.g., bg_oil_extraction)
type Resource struct {
	Type             string `json:"type"`
	UndiscoveredAmount int   `json:"undiscovered_amount"`
}

func main() {
	// Чтение файла
	file, err := os.Open("00_west_europe.txt")
	if err != nil {
		fmt.Printf("Ошибка открытия файла: %v\n", err)
		return
	}
	defer file.Close()

	// Парсинг
	states, err := parseStates(file)
	if err != nil {
		fmt.Printf("Ошибка парсинга: %v\n", err)
		return
	}

	// Сериализация в JSON
	jsonData, err := json.MarshalIndent(states, "", "  ")
	if err != nil {
		fmt.Printf("Ошибка сериализации в JSON: %v\n", err)
		return
	}

	// Вывод JSON (или сохранение в файл/БД)
	fmt.Println(string(jsonData))

	// Опционально: сохранение в файл
	err = os.WriteFile("states.json", jsonData, 0644)
	if err != nil {
		fmt.Printf("Ошибка записи JSON: %v\n", err)
	}
}

// parseStates парсит файл в []State
func parseStates(file *os.File) ([]State, error) {
	var states []State
	scanner := bufio.NewScanner(file)
	var currentState State
	var currentBlock string // Для отслеживания вложенных блоков
	inBlock := false
	listBuffer := make([]string, 0) // Для списков provinces, traits, etc.
	cappedResources := make(map[string]int)

	// Регулярные выражения
	keyValRegex := regexp.MustCompile(`^(\w+)\s*=\s*"?([^"{}\s][^"{}\n]*)"?$`)
	listStartRegex := regexp.MustCompile(`^(\w+)\s*=\s*{`)
	listItemRegex := regexp.MustCompile(`"([^"]+)"`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Начало блока STATE_XXX
		if strings.HasPrefix(line, "STATE_") {
			if inBlock && currentState.Name != "" {
				currentState.CappedResources = cappedResources
				states = append(states, currentState)
			}
			currentState = State{CappedResources: make(map[string]int)}
			currentBlock = ""
			inBlock = true
			parts := strings.Split(line, "=")
			currentState.Name = strings.TrimSpace(parts[0])
			continue
		}

		if !inBlock {
			continue
		}

		// Конец блока
		if line == "}" {
			if currentBlock == "capped_resources" {
				currentState.CappedResources = cappedResources
				cappedResources = make(map[string]int)
			} else if currentBlock == "provinces" {
				currentState.Provinces = listBuffer
				listBuffer = make([]string, 0)
			} else if currentBlock == "traits" {
				currentState.Traits = listBuffer
				listBuffer = make([]string, 0)
			} else if currentBlock == "impassable" {
				currentState.Impassable = listBuffer
				listBuffer = make([]string, 0)
			} else if currentBlock == "prime_land" {
				currentState.PrimeLand = listBuffer
				listBuffer = make([]string, 0)
			} else if currentBlock == "arable_resources" {
				currentState.ArableResources = listBuffer
				listBuffer = make([]string, 0)
			} else if currentBlock == "resource" {
				if len(listBuffer) >= 2 {
					amount, _ := strconv.Atoi(listBuffer[1])
					currentState.Resource = &Resource{
						Type:             listBuffer[0],
						UndiscoveredAmount: amount,
					}
				}
				listBuffer = make([]string, 0)
			} else {
				currentState.CappedResources = cappedResources
				states = append(states, currentState)
				currentState = State{CappedResources: make(map[string]int)}
				inBlock = false
			}
			currentBlock = ""
			continue
		}

		// Парсинг ключ-значение
		if matches := keyValRegex.FindStringSubmatch(line); len(matches) == 3 {
			key, value := matches[1], matches[2]
			switch currentBlock {
			case "capped_resources":
				if val, err := strconv.Atoi(value); err == nil {
					cappedResources[key] = val
				}
			case "resource":
				listBuffer = append(listBuffer, value)
			default:
				switch key {
				case "id":
					currentState.ID, _ = strconv.Atoi(value)
				case "subsistence_building":
					currentState.SubsistenceBuilding = value
				case "city":
					currentState.City = value
				case "port":
					currentState.Port = value
				case "farm":
					currentState.Farm = value
				case "mine":
					currentState.Mine = value
				case "wood":
					currentState.Wood = value
				case "arable_land":
					currentState.ArableLand, _ = strconv.Atoi(value)
				case "naval_exit_id":
					currentState.NavalExitID, _ = strconv.Atoi(value)
				}
			}
			continue
		}

		// Начало списка
		if matches := listStartRegex.FindStringSubmatch(line); len(matches) == 2 {
			currentBlock = matches[1]
			listBuffer = make([]string, 0)
			continue
		}

		// Элементы списка (provinces, traits, etc.)
		if currentBlock != "" {
			for _, match := range listItemRegex.FindAllStringSubmatch(line, -1) {
				listBuffer = append(listBuffer, match[1])
			}
		}
	}

	// Добавляем последний регион
	if inBlock && currentState.Name != "" {
		currentState.CappedResources = cappedResources
		states = append(states, currentState)
	}

	return states, scanner.Err()
}