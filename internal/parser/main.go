package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"states-shuffler/internal/types"
)

func main() {
	const inputDir = "D:\\Games\\Steam\\steamapps\\common\\Victoria 3\\game\\map_data\\state_regions"
	const outputJSON = "json/states.json"

	var states []types.State

	dirEntries, err := os.ReadDir(inputDir)
	if err != nil {
		fmt.Printf("Ошибка чтения директории %s: %v\n", inputDir, err)
		return
	}

	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() || dirEntry.Name() == "99_seas.txt" {
			continue
		}

		fileName := inputDir + "/" + dirEntry.Name()
		file, err := os.Open(fileName)
		if err != nil {
			fmt.Printf("Ошибка открытия файла %s: %v\n", fileName, err)
			continue // Продолжить со следующим файлом
		}

		states, err = parseStates(file, states)
		if err != nil {
			fmt.Printf("Ошибка парсинга %s: %v\n", fileName, err)
			continue
		}
	}

	if len(states) == 0 {
		fmt.Println("Не удалось распарсить ни одного региона")
		return
	}

	jsonData, err := json.MarshalIndent(states, "", "  ")
	if err != nil {
		fmt.Printf("Ошибка сериализации в JSON: %v\n", err)
		return
	}

	if err := os.MkdirAll("json", 0755); err != nil {
		fmt.Printf("Ошибка создания папки json: %v\n", err)
		return
	}

	if err := os.WriteFile(outputJSON, jsonData, 0644); err != nil {
		fmt.Printf("Ошибка записи JSON в %s: %v\n", outputJSON, err)
		return
	}

	fmt.Printf("Парсинг завершен! JSON сохранен в %s\n", outputJSON)
}

func parseStates(file *os.File, states []types.State) ([]types.State, error) {
	scanner := bufio.NewScanner(file)
	var currentState types.State
	var currentBlock string
	inBlock := false
	listBuffer := make([]string, 0)
	cappedResources := make(map[string]int)

	keyValRegex := regexp.MustCompile(`^(\w+)\s*=\s*"?([^"{}\s][^"{}\n]*)"?$`)
	listStartRegex := regexp.MustCompile(`^(\w+)\s*=\s*{`)
	listItemRegex := regexp.MustCompile(`"([^"]+)"`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "STATE_") {
			if inBlock && currentState.Name != "" {
				currentState.CappedResources = cappedResources
				states = append(states, currentState)
			}
			currentState = types.State{CappedResources: make(map[string]int)}
			currentBlock = ""
			inBlock = true
			parts := strings.Split(line, "=")
			currentState.Name = strings.TrimSpace(parts[0])
			continue
		}

		if !inBlock {
			continue
		}

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
					currentState.Resource = &types.Resource{
						Type:               listBuffer[0],
						UndiscoveredAmount: amount,
					}
				}
				listBuffer = make([]string, 0)
			} else {
				currentState.CappedResources = cappedResources
				states = append(states, currentState)
				currentState = types.State{CappedResources: make(map[string]int)}
				inBlock = false
			}
			currentBlock = ""
			continue
		}

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

		if matches := listStartRegex.FindStringSubmatch(line); len(matches) == 2 {
			currentBlock = matches[1]
			listBuffer = make([]string, 0)
			// Обработка списка в одной строке
			if strings.Contains(line, "}") {
				// Извлечь содержимое между { и }
				startIdx := strings.Index(line, "{") + 1
				endIdx := strings.Index(line, "}")
				if startIdx > 0 && endIdx > startIdx {
					listContent := strings.TrimSpace(line[startIdx:endIdx])
					for _, match := range listItemRegex.FindAllStringSubmatch(listContent, -1) {
						listBuffer = append(listBuffer, match[1])
					}
					switch currentBlock {
					case "provinces":
						currentState.Provinces = listBuffer
					case "traits":
						currentState.Traits = listBuffer
					case "impassable":
						currentState.Impassable = listBuffer
					case "prime_land":
						currentState.PrimeLand = listBuffer
					case "arable_resources":
						currentState.ArableResources = listBuffer
					}
					listBuffer = make([]string, 0)
					currentBlock = ""
				}
			}
			continue
		}

		if currentBlock != "" {
			for _, match := range listItemRegex.FindAllStringSubmatch(line, -1) {
				listBuffer = append(listBuffer, match[1])
			}
		}
	}

	if inBlock && currentState.Name != "" {
		currentState.CappedResources = cappedResources
		states = append(states, currentState)
	}

	return states, scanner.Err()
}
