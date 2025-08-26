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

// Основная функция парсера файлов Victoria 3 state_regions
// Проходит по директории с файлами регионов, парсит каждый файл
// и сохраняет результат в JSON формате для дальнейшей обработки
func main() {
	// Пути к исходным данным и результату
	const inputDir = "D:\\Games\\Steam\\steamapps\\common\\Victoria 3\\game\\map_data\\state_regions"
	const outputJSON = "json/states.json"

	// Карта для хранения результатов: ключ - имя файла, значение - массив состояний
	states := make(map[string][]types.State)

	// ШАГ 1: Чтение содержимого директории с файлами регионов Victoria 3
	dirEntries, err := os.ReadDir(inputDir)
	if err != nil {
		fmt.Printf("Ошибка чтения директории %s: %v\n", inputDir, err)
		return
	}

	// ШАГ 2: Обход всех файлов в директории state_regions
	for _, dirEntry := range dirEntries {
		// Пропускаем поддиректории и файл морей (не содержит обычных регионов)
		if dirEntry.IsDir() || dirEntry.Name() == "99_seas.txt" {
			continue
		}

		// ШАГ 3: Открытие каждого файла региона для парсинга
		fileName := inputDir + "/" + dirEntry.Name()
		file, err := os.Open(fileName)
		if err != nil {
			fmt.Printf("Ошибка открытия файла %s: %v\n", fileName, err)
			continue
		}

		// ШАГ 4: Парсинг содержимого файла и добавление к общим результатам
		states, err = parseStates(file, states, dirEntry.Name())
		if err != nil {
			fmt.Printf("Ошибка парсинга %s: %v\n", fileName, err)
			continue
		}
	}

	// Проверка что удалось что-то распарсить
	if len(states) == 0 {
		fmt.Println("Не удалось распарсить ни одного региона")
		return
	}

	// ШАГ 5: Сериализация всех собранных данных в JSON с форматированием
	jsonData, err := json.MarshalIndent(states, "", "  ")
	if err != nil {
		fmt.Printf("Ошибка сериализации в JSON: %v\n", err)
		return
	}

	// ШАГ 6: Создание директории для выходного файла
	if err := os.MkdirAll("json", 0755); err != nil {
		fmt.Printf("Ошибка создания папки json: %v\n", err)
		return
	}

	// ШАГ 7: Запись финального JSON файла
	if err := os.WriteFile(outputJSON, jsonData, 0644); err != nil {
		fmt.Printf("Ошибка записи JSON в %s: %v\n", outputJSON, err)
		return
	}

	fmt.Printf("Парсинг завершен! JSON сохранен в %s\n", outputJSON)
}

// parseStates - главная функция парсинга файлов Victoria 3
// Использует конечный автомат для анализа иерархической структуры файлов
// и извлечения данных о состояниях (регионах) игры
func parseStates(file *os.File, states map[string][]types.State, fileName string) (map[string][]types.State, error) {
	// Инициализация сканера для построчного чтения
	scanner := bufio.NewScanner(file)

	// === ПЕРЕМЕННЫЕ СОСТОЯНИЯ ПАРСЕРА ===
	var currentState types.State            // Текущее обрабатываемое состояние
	var currentBlock string                 // Имя активного подблока (provinces, traits, etc.)
	inBlock := false                        // Флаг: находимся ли внутри блока STATE_*
	listBuffer := make([]string, 0)         // Буфер для накопления элементов списков
	cappedResources := make(map[string]int) // Специальный буфер для ограниченных ресурсов

	// === РЕГУЛЯРНЫЕ ВЫРАЖЕНИЯ ДЛЯ ПАРСИНГА ===
	// Паттерн для строк типа: key = value или key = "value"
	keyValRegex := regexp.MustCompile(`^(\w+)\s*=\s*"?([^"{}\s][^"{}\n]*)"?$`)
	// Паттерн для начала блока-списка: key = {
	listStartRegex := regexp.MustCompile(`^(\w+)\s*=\s*{`)
	// Паттерн для элементов в кавычках внутри списков: "элемент"
	listItemRegex := regexp.MustCompile(`"([^"]+)"`)

	// === ОСНОВНОЙ ЦИКЛ ПАРСИНГА ===
	// Обрабатываем файл построчно, анализируя каждую строку
	for scanner.Scan() {
		line := scanner.Text()

		// ЭТАП 1: Предварительная обработка строки
		// Удаление UTF-8 BOM (Byte Order Mark), который может присутствовать в начале файла
		if strings.HasPrefix(line, "\ufeff") {
			line = strings.TrimPrefix(line, "\ufeff")
		}
		line = strings.TrimSpace(line)

		// Пропускаем пустые строки и комментарии (строки начинающиеся с #)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// ЭТАП 2: Обнаружение начала нового блока состояния
		// Блоки состояний всегда начинаются с префикса "STATE_"
		if strings.HasPrefix(line, "STATE_") {
			// Если мы уже обрабатывали предыдущее состояние - завершаем его
			if inBlock && currentState.Name != "" {
				currentState.CappedResources = cappedResources
				states[fileName] = append(states[fileName], currentState)
			}

			// Инициализация нового состояния
			currentState = types.State{CappedResources: make(map[string]int)}
			currentBlock = ""
			inBlock = true

			// Извлечение имени состояния (левая часть до знака "=")
			parts := strings.Split(line, "=")
			currentState.Name = strings.TrimSpace(parts[0])
			continue
		}

		// Если мы не внутри блока состояния - игнорируем строку
		if !inBlock {
			continue
		}

		// ЭТАП 3: Обработка закрывающих фигурных скобок "}"
		// Закрывающая скобка может означать конец подблока или конец всего состояния
		if line == "}" {
			// Определяем какой именно блок закрывается и выполняем соответствующие действия
			if currentBlock == "capped_resources" {
				// Завершение блока ограниченных ресурсов
				currentState.CappedResources = cappedResources
				cappedResources = make(map[string]int)
			} else if currentBlock == "provinces" {
				// Завершение списка провинций
				currentState.Provinces = listBuffer
				listBuffer = make([]string, 0)
			} else if currentBlock == "traits" {
				// Завершение списка особенностей региона
				currentState.Traits = listBuffer
				listBuffer = make([]string, 0)
			} else if currentBlock == "impassable" {
				// Завершение списка непроходимых территорий
				currentState.Impassable = listBuffer
				listBuffer = make([]string, 0)
			} else if currentBlock == "prime_land" {
				// Завершение списка плодородных земель
				currentState.PrimeLand = listBuffer
				listBuffer = make([]string, 0)
			} else if currentBlock == "arable_resources" {
				// Завершение списка сельскохозяйственных ресурсов
				currentState.ArableResources = listBuffer
				listBuffer = make([]string, 0)
			} else if currentBlock == "resource" {
				// Специальная обработка блока resource (содержит тип и количество)
				if len(listBuffer) >= 2 {
					amount, _ := strconv.Atoi(listBuffer[1])
					currentState.Resource = &types.Resource{
						Type:               listBuffer[0],
						UndiscoveredAmount: amount,
					}
				}
				listBuffer = make([]string, 0)
			} else {
				// Если currentBlock пустой - это конец всего блока состояния
				currentState.CappedResources = cappedResources
				states[fileName] = append(states[fileName], currentState)
				currentState = types.State{CappedResources: make(map[string]int)}
				inBlock = false
			}
			currentBlock = ""
			continue
		}

		// ЭТАП 4: Парсинг строк формата "key = value"
		if matches := keyValRegex.FindStringSubmatch(line); len(matches) == 3 {
			key, value := matches[1], matches[2]

			// Поведение зависит от того, в каком подблоке мы находимся
			switch currentBlock {
			case "capped_resources":
				// В блоке ограниченных ресурсов все значения - числовые лимиты
				if val, err := strconv.Atoi(value); err == nil {
					cappedResources[key] = val
				}
			case "resource":
				// В блоке resource накапливаем значения (тип ресурса и количество)
				listBuffer = append(listBuffer, value)
			default:
				// Вне подблоков - это основные поля состояния
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

		// ЭТАП 5: Обработка начала блоков-списков "key = {"
		if matches := listStartRegex.FindStringSubmatch(line); len(matches) == 2 {
			currentBlock = matches[1]
			listBuffer = make([]string, 0)

			// СПЕЦИАЛЬНЫЙ СЛУЧАЙ: Список помещается в одну строку "key = { "item1" "item2" }"
			if strings.Contains(line, "}") {
				// Извлекаем содержимое между фигурными скобками
				startIdx := strings.Index(line, "{") + 1
				endIdx := strings.Index(line, "}")
				if startIdx > 0 && endIdx > startIdx {
					listContent := strings.TrimSpace(line[startIdx:endIdx])

					// Ищем все элементы в кавычках и добавляем их в буфер
					for _, match := range listItemRegex.FindAllStringSubmatch(listContent, -1) {
						listBuffer = append(listBuffer, match[1])
					}

					// Применяем собранный список к соответствующему полю состояния
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

					// Сбрасываем состояние после обработки однострочного списка
					listBuffer = make([]string, 0)
					currentBlock = ""
				}
			}
			continue
		}

		// ЭТАП 6: Накопление элементов многострочных списков
		// Если мы внутри подблока-списка, ищем элементы в кавычках на текущей строке
		if currentBlock != "" {
			for _, match := range listItemRegex.FindAllStringSubmatch(line, -1) {
				listBuffer = append(listBuffer, match[1])
			}
		}
	}

	// ЭТАП 7: Финальная обработка
	// Сохраняем последнее состояние, если файл закончился без явной закрывающей скобки
	if inBlock && currentState.Name != "" {
		currentState.CappedResources = cappedResources
		states[fileName] = append(states[fileName], currentState)
	}

	// Возвращаем обновленную карту состояний и возможные ошибки сканирования
	return states, scanner.Err()
}
