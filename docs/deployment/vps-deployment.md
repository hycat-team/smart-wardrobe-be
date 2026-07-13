# VPS Deployment Guide

Documentation for deploying the Closy application to a VPS server.

## 1. Server Prerequisites
*   Operating System: Ubuntu 22.04 LTS or newer.
*   Pre-installed: Docker Engine and Docker Compose.
*   Domain configuration pointing to the VPS IP (e.g., `api.[DOMAIN].com`).

## 2. Installation Directory on VPS
The deployment source code and configuration files are located in the directory:
```bash
/opt/closy
```

## 3. Run the Application
Run the compose command on the VPS server:
```bash
docker compose --env-file .env.production -f docker-compose.prod.yml up -d
```

## 4. Nginx Configuration & Auto-Renew SSL (Certbot)
*   Use Certbot to request a free SSL certificate from Let's Encrypt.
*   **SSL Auto-Renew**: Set up a weekly cronjob to automatically renew the certificate:
    ```bash
    0 0 * * * certbot renew --post-hook "docker exec nginx-container nginx -s reload"
    ```
    *(Replace `nginx-container` with your actual Nginx container name).*
