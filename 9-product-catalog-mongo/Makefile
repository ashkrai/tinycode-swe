.PHONY: build run test docker-up docker-down lint seed

build:
	go build -o bin/api ./cmd/api

run: build
	MONGO_URI=mongodb://localhost:27017 PORT=8081 ./bin/api

test:
	go test ./... -v -count=1

docker-up:
	docker-compose up --build -d

docker-down:
	docker-compose down

seed:
	@echo "Seeding sample products..."
	@for i in 1 2 3; do \
		curl -s -X POST http://localhost:8081/products/ \
		  -H 'Content-Type: application/json' \
		  -d '{"name":"Laptop '$$i'","category":"electronics","price":999.99,"stock":50,"tags":["sale","featured"],"attributes":{"brand":"TechCo","warranty":"2yr"}}'; \
		echo ""; \
	done
	@for i in 1 2; do \
		curl -s -X POST http://localhost:8081/products/ \
		  -H 'Content-Type: application/json' \
		  -d '{"name":"Hammer '$$i'","category":"tools","price":24.99,"stock":200,"tags":["hardware"],"attributes":{"material":"steel"}}'; \
		echo ""; \
	done
	@curl -s -X POST http://localhost:8081/products/ \
	  -H 'Content-Type: application/json' \
	  -d '{"name":"Coffee Maker","category":"appliances","price":79.99,"stock":30,"tags":["kitchen"],"attributes":{"capacity":"12cup"}}'; \
	echo ""
	@echo "Done. Run: curl http://localhost:8081/products/analytics/categories"

lint:
	golangci-lint run ./...
