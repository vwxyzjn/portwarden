## The folloiwng commands should be executed separately.

# Build the production image
docker build -t vwxyzjn/portwarden-server-prod:1.7.0 -f Dockerfile.Build .

# Build the development image
docker build -t vwxyzjn/portwarden-server-dev:1.7.0 .

# minikube docker-env
