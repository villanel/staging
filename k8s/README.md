# Bookinfo Kubernetes Deployment

This directory contains Kubernetes configuration files for deploying the Bookinfo application.

## Architecture

The Bookinfo application consists of the following services:

- `productpage` - Main frontend service
- `reviews` - Reviews service
- `details` - Book details service  
- `ratings` - Ratings service
- `jaeger` - OpenTelemetry tracing backend

## Prerequisites

- Kubernetes cluster (local or remote)
- Docker for building images
- kubectl CLI configured to access the cluster

## Usage

### 1. Build Docker Images

First, build the Docker images for each service:

```bash
# Build all services
docker build -t bookinfo/ratings:latest -f ratings/Dockerfile.ratings .
docker build -t bookinfo/details:latest -f details/Dockerfile.details .
docker build -t bookinfo/reviews:latest -f reviews/Dockerfile.reviews .
docker build -t bookinfo/productpage:latest -f productpage/Dockerfile.productpage .
```

### 2. Deploy to Kubernetes

Apply the Kubernetes configuration:

```bash
# Apply all resources
kubectl apply -f k8s/bookinfo.yaml

# Verify deployment
kubectl get all -n bookinfo
```

### 3. Access Services

- **Productpage**: Access via NodePort at `http://<node-ip>:<node-port>/productpage/1`
  - Get the NodePort:
    ```bash
    kubectl get service productpage -n bookinfo -o jsonpath='{.spec.ports[0].nodePort}'
    ```

- **Jaeger UI**: Access via NodePort at `http://<node-ip>:<node-port>/`
  - Get the NodePort:
    ```bash
    kubectl get service jaeger -n bookinfo -o jsonpath='{.spec.ports[0].nodePort}'
    ```

### 4. Cleanup

To remove all resources:

```bash
kubectl delete -f k8s/bookinfo.yaml
```

## Resource Details

### Namespace
- **Name**: `bookinfo`
- **Purpose**: Isolates all Bookinfo resources

### Deployments
- **Replicas**: 1 per service
- **Images**: Local images with `imagePullPolicy: Never`

### Services

| Service | Type | Port | Description |
|---------|------|------|-------------|
| jaeger | ClusterIP | 16686 | Jaeger UI |
| jaeger | ClusterIP | 4317 | OTLP gRPC endpoint |
| ratings | ClusterIP | 9080 | Ratings service |
| details | ClusterIP | 9081 | Details service |
| reviews | ClusterIP | 9082 | Reviews service |
| productpage | NodePort | 9083 | Main frontend |

## Environment Variables

Each service is configured with appropriate environment variables:

- `OTEL_EXPORTER_OTLP_ENDPOINT`: Points to Jaeger for tracing
- Service-specific URLs (e.g., `RATINGS_SERVICE_URL`, `DETAILS_SERVICE_URL`)
