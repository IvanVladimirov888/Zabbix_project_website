// Использованные функции:
// document.addEventListener('DOMContentLoaded', function) – обработчик события загрузки DOM, инициализирует программу, проверяет наличие токена и перенаправляет пользователя на страницу входа при его отсутствии.
// getDevicesFromZabbix() – функция для получения списка устройств из Zabbix API, обработки данных и отображения их на странице.
// fetchTriggersAndUpdateIcon(device) – функция для получения триггеров устройства из Zabbix API, обновления иконки устройства в зависимости от приоритетов триггеров.
// getfillColor(priority) – функция для определения цвета иконки в зависимости от приоритета триггера.
// getDeviceIcon(device) – функция для получения пути к SVG-иконке устройства в зависимости от его типа.
// showDeviceInfo(hostID) – функция для отображения подробной информации об устройстве в боковой панели.
// extractContentWithoutBrackets(content) – функция для извлечения содержимого без HTML-тегов для сохранения в документе Word.
// saveDataAsWordFile(data) – функция для сохранения данных об устройстве в файл формата Word.
// getCookie(name) – функция для получения значения куки по имени.
// openSettings() – функция для открытия модального окна настроек.
// closeSettings() – функция для закрытия модального окна настроек.
// applySettings() – функция для применения настроек, таких как интервал обновления данных, и их сохранения.
// Инициализация программы:
// - Получение токена аутентификации из куки.
// - Перенаправление на страницу входа при отсутствии токена.
// - Получение списка устройств из Zabbix и обновление данных с заданным интервалом.
document.addEventListener('DOMContentLoaded', function () {
    const authToken = getCookie('zabbix_auth_token');                               // Получаем токен аутентификации из куки
    console.log('Auth Token:', authToken);

    if (!authToken) {                                                                     // Если токен отсутствует, то перенаправляем пользователя на страницу входа
        window.location.href = '/login.html';
        return;
    }
    // Функция получения списка устройств из Zabbix
    function getDevicesFromZabbix() {
        fetch('/api/devices', {
            method: 'GET',
            headers: {
                'Authorization': `Bearer ${authToken}`                                    // Установка заголовка авторизации с токеном
            }
        })
            .then(response => {
                if (!response.ok) {
                    console.log("Данные",response);
                    throw new Error('Ошибка при получении данных из Zabbix API');
                }
                return response.json();                                                   // Парсим ответ как Json
            })

            .then(devices => {
                console.log('Устройства из API:', devices);                               // Проверка того, что возвращает API

                const deviceList = document.getElementById('device-list');
                deviceList.innerHTML = '';                                                // Очистка списка устройств перед добавлением

                devices.forEach(device => {                                               // Перебор устройств перед добавлением в DOM
                    const deviceItem = document.createElement('div');
                    deviceItem.className = 'device-item';
                    deviceItem.id = device.hostid;

                    const deviceIconContainer = document.createElement('div');
                    deviceIconContainer.className = 'device-icon';
                    deviceIconContainer.id = `device-icon-${device.hostid}`;
                    deviceIconContainer.style.width = "50px"
                    deviceIconContainer.style.height = "50px"

                    const deviceDetails = document.createElement('div');
                    deviceDetails.className = 'device-details';
                    deviceDetails.innerHTML = `
                        <h4 style="margin-top: 5px; margin-bottom: 5px;">${device.host}</h4>
                        <p style="margin-top: 10px; margin-bottom: 2px;">ID: ${device.hostid}</p>
                        <p style="margin-top: 2px; margin-bottom: 2px;">IP: ${device.interfaces[0].ip}</p>`;

                    deviceItem.appendChild(deviceIconContainer);
                    deviceItem.appendChild(deviceDetails);
                    deviceList.appendChild(deviceItem);

                    fetchTriggersAndUpdateIcon(device,deviceIconContainer);

                    deviceItem.addEventListener('click', () => {
                        showDeviceInfo(device.hostid);
                    });
                });
            })
            .catch(error => {
                console.error('Ошибка загрузки устройства:', error);
            });
    }

    // Логика обработки и получения триггеров устройства
    function fetchTriggersAndUpdateIcon(device){
        fetch(`/api/devices/triggers?hostid=${device.hostid}`)
            .then(response => {
                if (!response.ok){
                    throw new Error("Ошибка при получении триггеров устройств");
                }
                return response.text()
            })
            .then(text =>{
                if(!text){
                    return; // Если ответ пустой то и текст пустой
                }
                return JSON.parse(text); //Если текст не пустой парсим его (JSON)
            })
            .then(triggers => {

                if (!Array.isArray(triggers)){
                    console.error("Неверный формат данных триггеров");
                    return
                }
                let maxPriority = 0;
                let activeTriggers = [];

                triggers.forEach(trigger => {
                    if (trigger.priority > maxPriority) {
                        maxPriority = trigger.priority;
                    }
                    if (trigger.status === "0") {
                        activeTriggers.push(trigger);
                    }
                });

                const fillColor = getfillColor(maxPriority);
                const iconPath = getDeviceIcon(device);
                const deviceIconContainer = document.getElementById(`device-icon-${device.hostid}`);
                if(deviceIconContainer) {
                    fetch(iconPath)
                        .then(response => response.text())
                        .then(svgText => {
                            deviceIconContainer.innerHTML = svgText;
                            const svgElement = deviceIconContainer.querySelector('svg');
                            if(svgElement){
                                svgElement.setAttribute('width','50');
                                svgElement.setAttribute('height','50');
                                const paths = svgElement.querySelectorAll('.bg');
                                paths.forEach(path => {
                                    path.style.fill = fillColor;
                                });

                                deviceIconContainer.dataset.activeTriggers = JSON.stringify(activeTriggers);

                            } else {
                                console.error("Не удалось получить документ SVG для устройства:",device.hostid);
                            }
                        })
                        .catch(error => {
                            console.error("Ошибка при загрузке SVG:",error);
                        });
                } else {
                    console.error("Элемент устройства не найден для пути:",iconPath);
                }
            })
            .catch(error => {
                console.error("Ошибка загрузки триггеров устройства:", error);
            });
    }
    function getfillColor(priority){
        if (priority === 0) {
            return  "#00b050";
        } else if (priority <= 3) {
            return "#fee599";
        } else {
            return "#c05046";
        }
    }
    const PCsvg = "/PC.svg"
    const Switchsvg = "/Switch.svg"
    const Serversvg = "/Server.svg"
    const Defaultsvg = "/Default.svg"


    // Функция получения иконки устройства в зависимотси от типа
    function getDeviceIcon(device) {
        if(!device.groups){
            console.log("Группы отсутсвуют для устройства", device)
            console.log("Название группы:", device.groups[0].name)
            return Defaultsvg
        }

        const groupNames = device.groups.map(group => device.groups[0].name.toLowerCase());
        console.log("Группа:", groupNames)
        if (groupNames.includes('arm')){
            return PCsvg;
        } else if (groupNames.includes('switch')){
            return Switchsvg;
        } else if (groupNames.includes('zabbix servers')){
            return Serversvg;
        }else return Defaultsvg;
    }



    function showDeviceInfo(hostID) {
        const sidebar = document.querySelector('.sidebar');
        sidebar.classList.add('open');
        console.log(hostID)

        fetch(`/api/deviceinfo?hostid=${hostID}`)
            .then(response => {
                if (!response.ok) {
                    throw new Error("Ошибка получения данных устройства");
                }
                return response.json();
            })
            .then(device => {
                console.log("Данные устройств:",device)
                const modalContent = document.getElementById('device-details-content');

                if (modalContent){
                    const deviceIconContainer = document.getElementById(`device-icon-${hostID}`);
                    console.log("DeviceIconContainer:",deviceIconContainer);
                    if (deviceIconContainer && deviceIconContainer.dataset.activeTriggers) {
                        let activeTriggers = JSON.parse(deviceIconContainer.dataset.activeTriggers);
                        let triggerInfo = '';
                        if (activeTriggers.length > 0) {
                            triggerInfo = '<h4>Активные триггеры:</h4><ul>';
                            activeTriggers.forEach(trigger => {
                                trigger.description = trigger.description.replace('{HOST.NAME}',device.host)
                                triggerInfo += `<li style="text-align: start; margin-top: 15px; font-size: 16pt;">${trigger.description} (приоритет: ${trigger.priority})</li>`;
                            });
                            triggerInfo += '</ul>';
                        } else {
                            triggerInfo = '<p>Нет активных триггеров.</p>';
                        }

                        modalContent.innerHTML = `
                        <h3 style="text-align: center">Детали устройства</h3>
                        <ul>
                            <li style="text-align: start; margin-top: 15px; font-size: 16pt;"> Имя хоста: "<h style="text-align: start;font-weight:bolder; font-size: 16pt">${device.hostName || "N/A"}<h/>"</li>
                            <li style="text-align: start; margin-top: 15px; font-size: 16pt;"> Доступная память: <h style="text-align: start;font-weight:bolder; font-family: 'Arial, sans-serif'; font-size: 16pt" >${device.availableMemory || "N/A"}<h/></li>
                            <li style="text-align: start; margin-top: 15px; font-size: 16pt;"> CPU в режиме ожидания: <h style="text-align: start;font-weight:bolder; font-size: 16pt" >${device.cpuIdleTime || "N/A"}<h/></li>
                            <li style="text-align: start; margin-top: 15px; font-size: 16pt;"> Свободное дисковое пространство: <h style="text-align: start;font-weight:bolder; font-size: 16pt" >${device.freeDiskSpace || "N/A"}<h/></li>
                            <li style="text-align: start; margin-top: 15px; font-size: 16pt;"> Общая память: <h style="text-align: start;font-weight:bolder; font-size: 16pt" >${device.totalMemory || "N/A"}<h/></li>
                            <li style="text-align: start; margin-top: 15px; font-size: 16pt;"> Общее пространство: <h style="text-align: start;font-weight:bolder; font-size: 16pt" >${device.totalSwapSpace || "N/A"}<h/></li>
                            <li style="text-align: start; margin-top: 15px; font-size: 16pt;"> Используемое дисковое пространство: <h style="text-align: start;font-weight:bolder; font-size: 16pt" >${device.usedDiskSpace || "N/A"}<h/></li>
                            <li style="text-align: start; margin-top: 15px; font-size: 16pt;"> Общее дисковое пространство: <h style="text-align: start;font-weight:bolder; font-size: 16pt" >${device.totalDiskSpace || "N/A"}<h/></li>
                            <li style="text-align: start; margin-top: 20px; font-size: 16pt;"> Информация о системе: <br><p style="font-weight:bolder; margin-top: 2px; font-size: 16pt" >${device.systemInformation || "N/A"}<p/></li>
                        </ul>
                        ${triggerInfo}`;
                    }
                    else {
                        console.error("Не удалось найти SVG элемент для устройства или данные триггеров не доступны:",hostID)
                    }
                } else {
                    console.error("Элемент с таким ID не найден");
                }
            })
            .catch(error => {
                console.error("Ошибка загрузки данных устройства:", error);
            });
    };

    const saveButton = document.getElementById('save-button');
    saveButton.addEventListener('click',()=>{
        const  modalContent = document.getElementById('device-details-content');
        if(modalContent){
            const  data = extractContentWithoutBrackets(modalContent.innerHTML);
            saveDataAsWordFile(data);
        }
        else {
            alert('Нет данных для сохранения');
        }
    });

    function extractContentWithoutBrackets(content){
        let text = content.replace(/<li[^>]*>|<\/li>|<h3[^>]*>|<\/h3>|<p[^>]*>|<\/p>|[{}[\]]|<h[^>]*>|<\/h>|<ul[^>]*>|<\/ul>|<br[^>]*>|<h4[^>]*>|<\/h4>|/g,'');
        text = text.trim();
        text = text.split('\n').map(line => line.trim()).join('\n');
        return text
    }
    function saveDataAsWordFile(data){
        const header = `
        <!DOCTYPE html>
        <html xmlns=":http://www.w3.org/1999/xhtml"
        <html lang="ru">
        <head>
            <meta charset="UTF-8">
            <title>Итоговый документ</title>
        <div style="text-align: center; font-weight: bold; font-size: 16px; font-family: 'Times New Roman';">
            Итоговый документ
            </div>
            <div style"height: 24pt;"></div>
            <div style = "font-size: 13px; font-family: 'Times New Roman';">
                ${data.replace(/\n/g,'<br>')}
            </div>
        `;
        const blob = new Blob([header], {type: 'application/msword'});
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'DeviceData.doc';
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
    }

    // Обработчик закрытия боковой панели
    const closeSidebarButton = document.querySelector('.close-button');
    closeSidebarButton.addEventListener('click', () => {
        const sidebar = document.querySelector('.sidebar')
        sidebar.classList.remove('open');
    });

    // Функция получения значения куки
    function getCookie(name) {
        const value = `; ${document.cookie}`;
        const parts = value.split(`; ${name}=`);
        if (parts.length === 2) return parts.pop().split(';').shift();
    }

    let refreshInterval = 30;
    let  refreshIntervalId;

    const settingsIcon = document.getElementById('settings-icon')
    const closeSettingsBtn = document.getElementById('close-settings')
    const applySettingsBtn = document.getElementById('apply-settings')

    if (settingsIcon && closeSettingsBtn && applySettingsBtn){
        settingsIcon.addEventListener("click", openSettings);
        closeSettingsBtn.addEventListener("click", closeSettings)
        applySettingsBtn.addEventListener("click",applySettings)
    }
    else {
        console.error("Элементы настройки не найдены")
    }

    function openSettings(){
        document.getElementById("settings-modal").style.display = "block"
    }

    function closeSettings(){
        document.getElementById("settings-modal").style.display = "none"
    }

    function applySettings(){
        console.log("Применение настроек")
        const refreshInput = document.getElementById("refresh-interval");
        const newInterval = parseInt(refreshInput.value, 10);

        if (!isNaN(newInterval) && newInterval > 0) {
            refreshInterval = newInterval;
            clearInterval(refreshIntervalId);
            refreshIntervalId = setInterval(getDevicesFromZabbix, refreshInterval * 1000);
        }

        closeSettings();
    }

    refreshIntervalId = setInterval(getDevicesFromZabbix, refreshInterval * 1000);
    getDevicesFromZabbix();
});
