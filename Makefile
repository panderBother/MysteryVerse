.PHONY: install dev build test lint format check docker-up docker-down smoke structure contracts-generate contracts-check db-generate db-check db-migrate db-status db-rollback db-test

install:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/mv.ps1 install

dev:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/mv.ps1 dev

build:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/mv.ps1 build

test:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/mv.ps1 test

lint:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/mv.ps1 lint

format:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/mv.ps1 format

check:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/mv.ps1 check

docker-up:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/mv.ps1 docker-up

docker-down:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/mv.ps1 docker-down

smoke:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/mv.ps1 smoke

structure:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/mv.ps1 structure

contracts-generate:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/contracts.ps1 generate

contracts-check:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/contracts.ps1 check

db-generate:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database.ps1 generate

db-check:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database.ps1 check

db-migrate:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database.ps1 migrate-up

db-status:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database.ps1 migrate-status

db-rollback:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database.ps1 migrate-down

db-test:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database.ps1 test
