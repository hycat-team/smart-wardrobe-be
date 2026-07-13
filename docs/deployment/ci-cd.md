# CI/CD Setup Guide

The project uses GitHub Actions as the automated integration and deployment system.

## 1. Secrets to Configure on the GitHub Repository
To ensure security, all sensitive information of the VPS server is stored as GitHub Repository Secrets:

*   `SSH_PRIVATE_KEY`: The SSH Private Key used to access the VPS.
*   `VPS_HOST`: The IP address of the VPS server (e.g., `[VPS_IP_PLACEHOLDER]`).
*   `VPS_USER`: The SSH account name to log into the server (e.g., `root`).
*   `CLOUDINARY_URL`, `DATABASE_URL`, `RABBITMQ_URL`... and other application environment configuration keys.

## 2. CI/CD Workflow
*   **CI**: Automatically triggers when there is a PR to the `develop` or `main` branch. Runs formatting checks, tests, and a test build.
*   **CD**: Automatically triggers when merging a PR into the `main` branch. Builds the Docker Image, pushes it to the Docker Registry, and connects via SSH for automated deployment on the VPS.
