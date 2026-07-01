# Database Design

The system uses a PostgreSQL relational database as the Single Source of Truth, integrating with GORM for entity management.

## 1. Core Tables

- `users`: Stores B2C user account information.
- `wardrobe_items`: Manages digitized clothing and image paths.
- `outfits`: Stores saved outfits.
- `brands`: Information about partner brands.
- `brand_members`: Manages employee roles belonging to brands.
- `loyalty_points`: Accumulated points of users at each brand.
- `campaigns`: Marketing events and campaigns of brands.
- `digital_samples`: Stores prototype designs in the Sample Lab.
- `sample_votes`: User voting history for design prototypes.
