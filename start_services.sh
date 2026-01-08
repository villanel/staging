#!/bin/bash

# Start all services in background
echo "Starting ratings service..."
go run ratings/main.go &
echo "Starting details service..."
go run details/main.go &
echo "Starting reviews service..."
go run reviews/main.go &
echo "Starting productpage service..."
go run productpage/main.go &

echo "All services started!"
echo "Productpage: http://localhost:9083/productpage/1"
echo "Details: http://localhost:9081/details/1"
echo "Reviews: http://localhost:9082/reviews/1"
echo "Ratings: http://localhost:9080/ratings/1"

echo "Press Ctrl+C to stop all services..."
wait