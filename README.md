# Профильное задание от VK

---
- [Установка и запуск](#установка-и-запуск)
- [Краткое описание](#краткое-описание)

## Установка и запуск

### Предварительные установки
На системе должны быть установлены 
> **git** 

> **docker engine** 

> **docker compose plugin**

Далее создаём рабочую директорию программы при помощи:
```
git clone https://github.com/Klimentin0/bot
```
Как директория будет создана далее переходим в корневую директорию проекта:
```
cd bot
```
И запускаем проект
```
docker compose up --build

```
Возможно нужны привилегии прм запуске
```
sudo docker compose up --build
```
Отображение в браузере происходит на:
> http://localhost:8065/

Для завершения
```
sudo docker compose down
```
