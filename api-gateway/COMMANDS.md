# Check container status
```docker ps```

# View container logs
```docker logs nginx-nginx-gateway-1```

# Test NGINX configuration
```docker exec nginx-nginx-gateway-1 nginx -t```

# Check if NGINX is responding
curl http://localhost


# Build the image
```docker build -t api-gateway```

# Run the container
```docker run -p 8080:8080 api-gateway```

# Or use with docker-compose
```docker-compose up```

1. For development with NGINX only:
```
cd api-gateway/docker/nginx
docker-compose -f docker-compose.nginx.yml up -d
```
2. For full stack development:
```
cd api-gateway/docker
docker-compose up -d
```
3. To test NGINX configuration:
```
# When running NGINX only
docker-compose -f docker/nginx/docker-compose.nginx.yml exec nginx-gateway nginx -t

# When running full stack
docker-compose exec nginx-gateway nginx -t
```

# Build and run the Docker container:

### Build the container
```docker-compose build```

### Start the container
```docker-compose up -d```

### Test NGINX configuration inside the container
```docker-compose exec api-gateway nginx -t```

### View logs
```docker-compose logs api-gateway```
