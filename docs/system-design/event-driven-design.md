# Event-Driven Design

Event-driven architecture helps asynchronously process AI tasks that consume a lot of time and resources, ensuring the API Gateway always responds quickly.

## 1. Events

| Event | Publisher | Consumer | Routing Key | Payload |
|-------|-----------|----------|-------------|---------|
| `wardrobe.event.created` | wardrobe module | elasticsearch sync | `wardrobe.event.*` | WardrobeEventPayload |
| `wardrobe.event.updated` | wardrobe module | elasticsearch sync | `wardrobe.event.*` | WardrobeEventPayload |
| `wardrobe.event.deleted` | wardrobe module | elasticsearch sync | `wardrobe.event.*` | WardrobeEventPayload |
| `fashion.event.init` | wardrobe module | fashion init worker | `fashion.event.init` | FashionAnalyzeJobDTO |
| `fashion.event.analyze` | fashion module | fashion analyze worker | `fashion.event.analyze` | FashionAnalyzeJobDTO |
| `subscription.plan.changed` | subscription module | aiexecutor plan changed worker | `subscription.plan.changed` | PlanChangedPayload |

## 2. Operating Mechanism with RabbitMQ

- When a user uploads an item of clothing (`wardrobe.event.created`), the wardrobe module saves the item and pushes the event into RabbitMQ.
- The elasticsearch sync worker consumes `wardrobe.event.*` events to keep the search index up to date.
- The fashion.init worker consumes `fashion.event.init` to run AI analysis (background removal, label extraction) on new items.
- The fashion.analyze worker consumes `fashion.event.analyze` for reprocessing or retry of failed analyses.
- When a subscription plan changes (`subscription.plan.changed`), the aiexecutor plan changed worker creates/updates the user's AI policy grant.
- The AI usage reconciliation worker (cron-based, not event-driven) periodically expires unknown/reserved usage events.

## 3. Architectural Advantages

- **Avoid timeouts**: The client does not have to wait for synchronous responses from slow AI models.
- **Ensure reliability**: Supports retry and dead-letter queue (DLQ) mechanisms when calling external AI services encounters temporary connection errors.
- **Modular boundaries**: Write operations across modules are pushed through events instead of direct function calls or GORM preloads.
