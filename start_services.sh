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
echo "Productpage: http://localhost:9083/productpage"
echo "Details: http://localhost:9081/details"
echo "Reviews: http://localhost:9082/reviews"
echo "Ratings: http://localhost:9080/ratings"

echo "Press Ctrl+C to stop all services..."
wait