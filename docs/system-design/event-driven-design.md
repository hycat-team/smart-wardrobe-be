# Event-Driven Design

Event-driven architecture helps asynchronously process AI tasks that consume a lot of time and resources, ensuring the API Gateway always responds quickly.

## 1. Operating Mechanism with RabbitMQ

- When a user uploads an item of clothing, the API saves temporary information and pushes a `wardrobe.item.uploaded` event into the RabbitMQ queue.
- The Event Consumer worker receives the job and sends the image via the AI API for background removal and label extraction.
- After receiving the results from AI, the Worker updates the Item status to `Active` and sends a notification via WebSocket to the user.

## 2. Architectural Advantages

- **Avoid timeouts**: The client does not have to wait for synchronous responses from slow AI models.
- **Ensure reliability**: Supports retry and dead-letter queue (DLQ) mechanisms when calling external AI services encounters temporary connection errors.
