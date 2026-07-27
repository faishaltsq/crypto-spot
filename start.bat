@echo off
if not exist .env copy .env.example .env >nul
docker compose up --build
