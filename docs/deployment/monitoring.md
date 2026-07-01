# System Monitoring

Monitor the health and performance of the Closy backend system.

## 1. Logging System
*   The system uses the `zap` library to record application info and error logs.
*   Logs are output to the console and saved to periodically rotated log files (log rotation) located in the log directory configured in the `.env` file.

## 2. API Health Check
*   API health monitoring path: `/api/v1/health`.
*   Configure a monitoring tool (Uptime Kuma or equivalent) to send notifications to the technical team's Slack/Telegram channel if this endpoint does not respond for more than 2 minutes.
