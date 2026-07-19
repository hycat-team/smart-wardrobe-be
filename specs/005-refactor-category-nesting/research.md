# Research & Decisions - Refactor Category Nesting

## Decisions

### 1. Nesting Category inside FashionItemRes
- **Decision**: Define a new struct `CategoryBriefRes` and embed it as `Category *CategoryBriefRes` inside `FashionItemRes` and `FashionItemBriefRes`.
- **Rationale**: Currently, DTOs flats category fields like `CategoryName` and `CategorySlug`. By nesting `CategoryBriefRes`, we keep the DTO design coherent and clean.
- **Alternatives Considered**: Keeping flat fields was rejected because it causes duplicate mapping and custom mapping logic across modules (e.g., brand and wardrobe modules had to map flat fields manually into their own custom category representations).

### 2. Eliminating Duplicate DTOs
- **Decision**: Remove local custom category/fashion structs from `wardrobe` and `brand` modules and use the shared `FashionItemRes` from the `fashion` module contract.
- **Rationale**: DTOs should not be duplicated across modules when they represent the exact same domain data boundary.
- **Alternatives Considered**: Keeping them was rejected because it introduces double definition and code maintenance overhead.
