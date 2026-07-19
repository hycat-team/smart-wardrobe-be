# Data Model & DTOs - Refactor Category Nesting

## DTO Structures

### 1. CategoryBriefRes (Shared in `fashion` module application DTOs)
Represents a lightweight, nested category object without database sorting fields.

| Field | Type | Description |
|---|---|---|
| `id` | `uuid.UUID` | Unique identifier of the category |
| `name` | `string` | Human-readable name (Vietnamese) |
| `slug` | `string` | Unique URL-friendly identifier |

### 2. FashionItemRes (Shared in `fashion` module contract)
Updated to include the nested `CategoryBriefRes`.

| Field | Type | Description |
|---|---|---|
| `id` | `uuid.UUID` | Unique identifier of the fashion item |
| `category` | `*CategoryBriefRes` | Nested category details (can be `nil`) |
| `imageUrl` | `string` | URL to the image resource |
| `color` | `*string` | Main color name |
| `colorHex` | `*string` | Hexadecimal color code |
| `colorHue` | `*float64` | HSL Hue value |
| `colorSaturation` | `*float64` | HSL Saturation value |
| `colorLightness` | `*float64` | HSL Lightness value |
| `style` | `*string` | Style category |
| `material` | `*string` | Material composition |
| `pattern` | `*string` | Fabric pattern |
| `fit` | `*string` | Fit type |
| `seasonality` | `*string` | Seasonality indicators |
| `description` | `*string` | Detailed description |
| `createdAt` | `time.Time` | Creation timestamp |
| `updatedAt` | `time.Time` | Modification timestamp |

*(Note: `FashionItemBriefRes` will also be updated to nest `Category *CategoryBriefRes` to maintain consistency).*
