// ПП ПО ПМ.02 - "Осуществление интегрции программных модулей"
// по теме: разработка программного комплекса контроля состояния технических средств
// Разработал: Владимиров Иван Сергеевич
// Группа: ТИП-62
// Дата и номер версии:  31.05.24
// Язык: Go
// Краткое описание: веб-сервер для обработки информации о состоянии технических средств и веб-интерфейс для отображения состояния технических средств.
//Задание:
//1. Анализ и разработка диаграмм.
//2. Разработка программного обеспечения.
//3. Формулировка спецификации.
//4. Заполнение и структурирование базы данных.
//5. Создание главной формы приложения.
//6. Реализация форм авторизации и регистрации с требованиями к паролям.
//7. Создание пользовательских форм для управления данными.
//8. Написание вычисляемых функций.
//9. Тестирование и написание отчета.
//10. Подготовка презентации.
//Использованные формы:
//login.html – форма для аутентификации пользователя;
//main.html – основная форма для взаимоействия с устройствами.
//Использованные файлы:
// script - фронт
//Использованные функции:
//getDeviceInfoFromZabbix(authToken, hostID string) – функция для выборки данных устройств с сервиса Zabbix;
//convert(b string) – функция конвертации байтов в гигобайты;
//getDevicesFromZabbix(authToken string) - функция для выборки устройств из zabbix;
//authenticateWithZabbix(username, password string) – функция аутентификации пользователя в zabbix и apache2;
//getTriggersFromZabbix(authToken, hostID string) – функция для выборки данных триггера из zabbix сервиса;
//fileServerHandler(w http.ResponseWriter, r *http.Request) - функция для подгружения файлов на наше веб-приложение.
//Данный файл является точкой старта приложения

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const (
	zabbixAPIURL = "http://192.168.42.20/zabbix/api_jsonrpc.php"
)

type Device struct {
	ID         string `json:"hostid"`
	Host       string `json:"host"`
	Interfaces []struct {
		Interfaceid string `json:"interfaceid"`
		IP          string `json:"ip"`
	} `json:"interfaces,omitempty"`
	Groups []struct {
		Groupid string `json:"groupid"`
		Name    string `json:"name"`
	} `json:"groups,omitempty"`
	HostName          string `json:"hostName"`
	SystemInformation string `json:"systemInformation"`
	TotalMemory       string `json:"totalMemory"`
	AvailableMemory   string `json:"availableMemory"`
	CPUIdleTime       string `json:"cpuIdleTime"`
	TotalSwapSpace    string `json:"totalSwapSpace"`
	UsedDiskSpace     string `json:"usedDiskSpace"`
	TotalDiskSpace    string `json:"totalDiskSpace"`
	FreeDiskSpace     string `json:"freeDiskSpace"`
}

func getDeviceInfoFromZabbix(authToken, hostID string) (Device, error) {
	requestData := fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"method": "item.get",
		"params": {
			"output": ["key_","name","lastvalue"],
			"hostids":"%s",
			"filter": {
				"key_": ["status","system.hostname","agent.hostname","hostid","system.host","system.uname","vm.memory.size[available]","system.cpu.util[,idle]","vfs.fs.size[/,free]","vm.memory.size[total]","system.swap.size[,total]","vfs.fs.size[\/,used]","vfs.fs.size[\/,total]"]
			},
			"sortfield": "key_"
		},
		"auth": "%s",
		"id": 1
	}`, hostID, authToken)

	log.Printf("Токен: %s", authToken)
	log.Printf("Запрос к Zabbix API для получения информации об устройстве: %s", requestData)

	req, err := http.NewRequest("POST", zabbixAPIURL, strings.NewReader(requestData))
	if err != nil {
		log.Printf("Ошибка при создании запроса: %v", err)
		return Device{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("pamuser", "12345678")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Ошибка при выполнении POST-запроса: %v", err)
		return Device{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Ошибка при чтении тела ответа: %v", err)
		return Device{}, err
	}
	log.Printf("Тело ответа от Zabbix API: %s", body)

	if resp.StatusCode != http.StatusOK {
		return Device{}, fmt.Errorf("получен неожиданный статус ответа от сервера: %d, тело ответа: %s", resp.StatusCode, string(body))
	}

	var data struct {
		Result []struct {
			ItemID    string `json:"itemid"`
			Name      string `json:"name"`
			Key       string `json:"key_"`
			LastValue string `json:"lastvalue"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		log.Printf("Ошибка декодирования ответа: %v", err)
		return Device{}, err
	}
	if data.Error != nil {
		log.Printf("Ошибка Zabbix API: %s (код: %d)", data.Error.Message, data.Error.Code)
		return Device{}, errors.New(data.Error.Message)
	}

	deviceInfo := Device{}
	for _, item := range data.Result {
		log.Printf(item.Key)
		switch item.Key {
		case "system.hostname":
			deviceInfo.HostName = item.LastValue
		case "vm.memory.size[available]":
			{
				availableMemory, err := convert(item.LastValue)
				if err != nil {
					log.Printf("Ошибка при конвертации размера доступной памяти: %v", err)
					availableMemory = "N/A"
				}
				deviceInfo.AvailableMemory = availableMemory + "GB"
				break
			}
		case "agent.hostname":
			deviceInfo.Host = item.LastValue
			break
		case "hostid":
			deviceInfo.ID = item.LastValue
			break
		case "system.cpu.util[,idle]":
			deviceInfo.CPUIdleTime = item.LastValue
			break

		case "vfs.fs.size[/,free]":
			{
				freeDiskSpace, err := convert(item.LastValue)
				if err != nil {
					log.Printf("Ошибка при конвертации размера доступной памяти: %v", err)
					freeDiskSpace = "N/A"
				}
				deviceInfo.FreeDiskSpace = freeDiskSpace + "GB"
				break
			}

		case "vfs.fs.size[/,used]":
			{
				usedDiskSpace, err := convert(item.LastValue)
				if err != nil {
					log.Printf("Ошибка при конвертации размера доступной памяти: %v", err)
					usedDiskSpace = "N/A"
				}
				deviceInfo.UsedDiskSpace = usedDiskSpace + "GB"
				break
			}

		case "vfs.fs.size[/,total]":
			{
				totalDiskSpace, err := convert(item.LastValue)
				if err != nil {
					log.Printf("Ошибка при конвертации размера доступной памяти: %v", err)
					totalDiskSpace = "N/A"
				}
				deviceInfo.TotalDiskSpace = totalDiskSpace + "GB"
				break
			}

		case "vm.memory.size[total]":
			{
				totalMemory, err := convert(item.LastValue)
				if err != nil {
					log.Printf("Ошибка при конвертации размера доступной памяти: %v", err)
					totalMemory = "N/A"
				}
				deviceInfo.TotalMemory = totalMemory + "GB"
				break
			}

		case "system.swap.size[,total]":
			{
				totalSwapSpace, err := convert(item.LastValue)
				if err != nil {
					log.Printf("Ошибка при конвертации размера доступной памяти: %v", err)
					totalSwapSpace = "N/A"
				}
				deviceInfo.TotalSwapSpace = totalSwapSpace + "GB"
				break
			}

		case "system.uname":
			deviceInfo.SystemInformation = item.LastValue
			break
		}
	}

	log.Printf("Успешно получена информация об устройстве из Zabbix: %v", deviceInfo)
	return deviceInfo, nil
}
func convert(b string) (string, error) {
	bInt, err := strconv.ParseFloat(b, 64)
	if err != nil {
		return "", err
	}
	g := bInt / (1024 * 1024 * 1024)
	return fmt.Sprintf("%.2f", g), nil
}

func getDevicesFromZabbix(authToken string) ([]Device, error) {
	requestData := `{
		"jsonrpc": "2.0",
		"method": "host.get",
		"params": {
			"output": ["hostid", "host","name","systeminfo","inventory"],
			"selectInterfaces": ["interfaceid","ip"],
			"selectGroups": ["groupid","name"],
			"selectItems": ["itemid","name","key_","lastvalue"]
		},
		"auth": "` + authToken + `",
		"id": 1
	}`
	log.Printf("Токен:" + authToken)
	log.Printf("Запрос к Zabbix API: %s" + requestData)

	req, err := http.NewRequest("POST", zabbixAPIURL, strings.NewReader(requestData))
	if err != nil {
		log.Printf("Ошибка при создании запроса: %v", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("pamuser", "12345678")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Ошибка при выполнении POST-запроса: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Ошибка при чтении тела ответа: %v", err)
		return nil, err
	}
	log.Printf("Тело ответа от Zabbix API: %s", body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("получен неожиданный статус ответа от сервера: %d, тело ответа: %s", resp.StatusCode, string(body))
	}

	var data struct {
		Result []Device `json:"result"`
		Error  *struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		log.Printf("Ошибка декодирования ответа: %v", err)
		return nil, err
	}
	if data.Error != nil {
		log.Printf("Ошибка Zabbix API: %s (код: %d)", data.Error.Message, data.Error.Code)
		return nil, errors.New(data.Error.Message)
	}
	log.Printf("Успешно получены устройства из Zabbix: %v", data.Result)
	return data.Result, nil
}

func authenticateWithZabbix(username, password string) (string, error) {
	requestData := `{
		"jsonrpc": "2.0",
		"method": "user.login",
		"params": {
			"user": "` + username + `",
			"password": "` + password + `"
		},
		"id": 1
	}`
	log.Printf("Запрос на авторизацию в Zabbix: %w", requestData)

	req, err := http.NewRequest("POST", zabbixAPIURL, strings.NewReader(requestData))
	if err != nil {
		log.Printf("Ошибка при создании запроса: %v", err)
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("pamuser", "12345678")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Ошибка при выполнении POST-запроса: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Ошибка при чтении тела ответа: %v", err)
		return "", err
	}
	log.Printf("Тело ответа от Zabbix API: %s", body)
	log.Printf("Получен ответ от Zabbix")
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("получен неожиданный статус ответа от сервера: %d, тело ответа: %s", resp.StatusCode, string(body))
	}

	var responseBody map[string]interface{}
	if err := json.Unmarshal(body, &responseBody); err != nil {
		log.Printf("Ошибка декодирования ответа: %v", err)
		return "", err
	}
	result, ok := responseBody["result"].(string)
	if !ok {
		log.Printf("Ошибка: поле 'result' не является строкой")
		return "", errors.New("поле 'result' не является строкой")
	}
	log.Printf("Успешная авторизация в Zabbix. Токен: %s", result)
	return result, nil
}

func fileServerHandler(w http.ResponseWriter, r *http.Request) {
	filePath := "C:/Users/User/Desktop/Ivan/16_day/static/" + r.URL.Path
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "Файл не найден", http.StatusNotFound)
		return
	}
	http.ServeFile(w, r, filePath)
}

func getTriggersFromZabbix(authToken, hostID string) ([]Trigger, error) {
	requestData := fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"method": "trigger.get",
		"params": {
			"output": ["triggerid", "description", "priority", "status", "lastchange"],
			"hostids": "%s",			
			"filter": {
				"value": 1
			}
		},
		"auth": "%s",
		"id": 1
	}`, hostID, authToken)

	log.Printf("Запрос к Zabbix API для получения триггеров: %s", requestData)

	req, err := http.NewRequest("POST", zabbixAPIURL, strings.NewReader(requestData))
	if err != nil {
		log.Printf("Ошибка при создании запроса: %v", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("pamuser", "12345678")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Ошибка при выполнении POST-запроса: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Ошибка при чтении тела ответа: %v", err)
		return nil, err
	}
	log.Printf("Тело ответа от Zabbix API: %s", body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("получен неожиданный статус ответа от сервера: %d, тело ответа: %s", resp.StatusCode, string(body))
	}

	var data struct {
		Result []Trigger `json:"result"`
		Error  *struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		log.Printf("Ошибка декодирования ответа: %v", err)
		return nil, err
	}
	if data.Error != nil {
		log.Printf("Ошибка Zabbix API: %s (код: %d)", data.Error.Message, data.Error.Code)
		return nil, errors.New(data.Error.Message)
	}

	activeTriggerCount := len(data.Result)
	log.Printf("Количество активных триггеров для hostID %s: %d", hostID, activeTriggerCount)
	//for _, trigger := range data.Result {
	//	log.Printf("Триггер: %s, Описание: %s, Приоритет: %s, Статус: %s, Последнее изменение: %s", trigger.TriggerID, trigger.Description, trigger.Priority, trigger.Status, trigger.LastChange)
	//}

	return data.Result, nil
}

type Trigger struct {
	TriggerID   string `json:"triggerid"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	Status      string `json:"status"`
	LastChange  string `json:"lastchange"`
}

func main() {
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "Неверный тип содержимого", http.StatusBadRequest)
			return
		}
		var creds struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			http.Error(w, "Ошибка при декодировании JSON", http.StatusBadRequest)
			return
		}

		zabbixAuthToken, err := authenticateWithZabbix(creds.Username, creds.Password)
		if err != nil {
			http.Error(w, "Ошибка аутентификации в Zabbix: "+err.Error(), http.StatusUnauthorized)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:  "zabbix_auth_token",
			Value: zabbixAuthToken,
			Path:  "/",
		})

		http.Redirect(w, r, "/main.html", http.StatusSeeOther)
	})
	http.HandleFunc("/api/devices/", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("zabbix_auth_token")
		if err != nil {
			http.Error(w, "Токен аутентификации не найден", http.StatusUnauthorized)
			return
		}
		authToken := cookie.Value

		devices, err := getDevicesFromZabbix(authToken)
		if err != nil {
			http.Error(w, "Ошибка при получении данных из Zabbix API: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		jsonData, err := json.Marshal(devices)
		if err != nil {
			http.Error(w, "Ошибка при кодировании данных в JSON: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(jsonData)
	})

	http.HandleFunc("/api/deviceinfo/", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("zabbix_auth_token")
		if err != nil {
			http.Error(w, "Токен аутентификации не найден", http.StatusUnauthorized)
			return
		}
		authToken := cookie.Value

		hostID := r.URL.Query().Get("hostid")
		if hostID == "" {
			http.Error(w, "hostID устройства не указан", http.StatusBadRequest)
			return
		}

		deviceInfo, err := getDeviceInfoFromZabbix(authToken, hostID)
		if err != nil {
			http.Error(w, "Ошибка при получении данных устройств из Zabbix API: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		jsonData, err := json.Marshal(deviceInfo)
		if err != nil {
			http.Error(w, "Ошибка при кодировании данных в JSON: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(jsonData)
	})

	http.HandleFunc("/api/devices/triggers/", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("zabbix_auth_token")
		if err != nil {
			http.Error(w, "Токен аутентификации не найден", http.StatusUnauthorized)
			return
		}
		authToken := cookie.Value
		hostID := r.URL.Query().Get("hostid")
		if hostID == "" {
			http.Error(w, "hostID устройства не указан", http.StatusBadRequest)
			return
		}
		triggers, err := getTriggersFromZabbix(authToken, hostID)
		if err != nil {
			http.Error(w, "Ошибка при получении триггеров из Zabbix API: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		jsonData, err := json.Marshal(triggers)
		if err != nil {
			http.Error(w, "Ошибка при кодировании данных в JSON: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(jsonData)
		log.Printf(string(jsonData))
	})

	http.HandleFunc("/PC.svg", fileServerHandler)
	http.HandleFunc("/Switch.svg", fileServerHandler)
	http.HandleFunc("/Server.svg", fileServerHandler)
	http.HandleFunc("/Default.svg", fileServerHandler)
	http.HandleFunc("/GearWheel.png", fileServerHandler)
	http.Handle("/", http.FileServer(http.Dir("C:/Users/User/Desktop/Ivan/16_day/static/")))

	if _, err := os.Stat("C://Users/User/Desktop/Ivan/16_day/static/login.html"); os.IsNotExist(err) {
		log.Fatal("Файл login.html не найден в каталоге")
	}

	log.Println("Запуск веб-сервера на порту: 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
