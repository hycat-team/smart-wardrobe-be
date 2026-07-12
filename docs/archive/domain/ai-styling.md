# Fashion Color Theory Engine Specification

## I. Color Space Conversion Algorithm from RGB to HSL

Initial clothing color data is usually stored or extracted as RGB with $R, G, B \in [0, 255]$. To evaluate human visual characteristics and calculate positions on the color wheel, the system converts this data into the HSL color space.

### 1. Input Normalization

Convert the three color channels from an 8-bit scale to a fractional scale:

$$R' = \frac{R}{255}$$

$$G' = \frac{G}{255}$$

$$B' = \frac{B}{255}$$

Determine the maximum, minimum, and difference values:

$$C_{max} = \max(R', G', B')$$

$$C_{min} = \min(R', G', B')$$

$$\Delta = C_{max} - C_{min}$$

### 2. Calculate Lightness ($L$)

Lightness is the average perception of the lightness or darkness of a color:

$$L = \frac{C_{max} + C_{min}}{2}$$

### 3. Calculate Saturation ($S$)

Saturation represents the purity of the color:

- **Achromatic case ($\Delta = 0$)**  
  $$S = 0$$

- **Chromatic case ($\Delta > 0$)**  
  $$
  S = \begin{cases}
  \frac{\Delta}{C_{max} + C_{min}} & \text{if } L \le 0.5 \\
  \frac{\Delta}{2 - (C_{max} + C_{min})} & \text{if } L > 0.5
  \end{cases}
  $$

### 4. Calculate Hue ($H$)

Hue is the angular position of the color on the 360-degree color wheel:

$$
H = \begin{cases}
0^\circ & \text{if } \Delta = 0 \\
60^\circ \times \left( \frac{G' - B'}{\Delta} \bmod 6 \right) & \text{if } C_{max} = R' \\
60^\circ \times \left( \frac{B' - R'}{\Delta} + 2 \right) & \text{if } C_{max} = G' \\
60^\circ \times \left( \frac{R' - G'}{\Delta} + 4 \right) & \text{if } C_{max} = B'
\end{cases}
$$

### Current Version

The current version of the system should not be understood as having the entire formula above being used as a complete geometric engine in every flow.

However, color data still plays a real role in the current backend:

- Extracted by AI when processing item images
- Stored as item metadata
- Used in rich text context to generate embeddings

In other words, the formula part is kept as the target theoretical foundation, while the current role of color is as an important input for AI and for advanced outfit coordination directions.

---

## II. Achromatic Color Filtering Pipeline

Before running paired matching according to geometric angles, the system separates neutral colors from colors with clear chromatic anchor points.

- An item is placed into the `Neutral_Pool` instead of the `Chroma_Pool` if it meets one of the following conditions:
  - **Black:** $L \le 10\%$
  - **White:** $L \ge 90\%$
  - **Gray:** $S \le 10\%$

### Target Design

This design helps the system:

- Avoid forcing neutral colors into unnecessary geometric rules
- Allow neutral items to combine more flexibly

### Current Version

The current version of the code should not be described as having fully implemented the `Neutral_Pool` or `Chroma_Pool` pipeline exactly as the old specification in every recommendation step.

However, this concept should still be kept in the docs because:

- It is a reasonable theoretical foundation for the outfit coordination system
- It can be reused in local swap, advanced recommendation, or local filters in the future

---

## III. Geometric Color Coordination Rules

For items with structured colors, the engine will calculate the angle difference on the color wheel:

$$\Delta H = |H_{\text{Item1}} - H_{\text{Item2}}|$$

$$\Delta H_{\text{final}} = \min(\Delta H, 360^\circ - \Delta H)$$

### 1. Analogous Color Combination

The goal is to create harmonious and low-contrast outfits.

- **Primary constraint:**

$$\Delta H_{\text{final}} < 30^\circ$$

- **Minimum Lightness/Darkness contrast constraint:**

$$|L_{\text{Item1}} - L_{\text{Item2}}| \ge 15\%$$

### 2. Complementary Color Combination

The goal is to create a strongly contrasting color scheme.

- **Primary constraint:**

$$165^\circ \le \Delta H_{\text{final}} \le 195^\circ$$

### Current Version

These rules are kept in the document as the **target theoretical engine** of the outfit coordination system.

In the current state:

- The backend already has AI recommendation
- The backend already has color metadata
- The backend already has descriptions in other documents about the color filter stage or style matrix

But it should not be overly asserted that all formulas here are running directly and fully in the entire current implementation if the code doesn't clearly show every step.

---

## IV. Output Data Interaction Diagram

After the backend completes calculating color combinations, the payload can be packaged into a standard schema for subsequent AI or aggregation steps.

```json
{
  "user_context": {
    "body_profile": "..."
  },
  "pre_validated_pairs": {
    "complementary_suggestions": [
      {
        "set_id": "pair_01",
        "top": {
          "id": "uuid-1",
          "name": "Orange T-shirt",
          "hex": "#FF5733",
          "hsl": [11, 100, 60]
        },
        "bottom": {
          "id": "uuid-2",
          "name": "Blue Jeans",
          "hex": "#2E4053",
          "hsl": [210, 28, 25]
        },
        "calculated_delta_hue": 199
      }
    ],
    "neutral_matchings": [
      {
        "top_id": "uuid-1",
        "neutral_bottom_id": "uuid-black-jeans"
      }
    ]
  }
}
```

### Target Output Direction based on Current Recommendation

When mapped with the current recommendation DTO, the color data layer and pre-validated pairs can be used as input to generate business output in the form of:

```json
{
  "title": "Casual weekend outfit",
  "explanation": "Combines neutral tones with light color accents, suitable for weather and user style",
  "items": [
    {
      "role": "top",
      "primary": { "id": "uuid-1" },
      "alternatives": [{ "id": "uuid-3" }]
    },
    {
      "role": "bottom",
      "primary": { "id": "uuid-2" },
      "alternatives": [{ "id": "uuid-4" }]
    }
  ]
}
```

This schema shows that theoretical color data does not necessarily need to go straight out as pure color pairs. Instead, it can be digested into:

- Outfit title
- Selection explanation
- Item groups by role
- Primary and alternative items for each role

### Current Version

The above schema is still a very useful description of the **target data** for outfit aggregation steps using AI or local filters.

In the current backend:

- Items already have color data and embeddings
- Rich text context uses color metadata
- The recommendation engine has the data foundation to gradually move towards a payload like above

Therefore, this section is not removed but understood as the target data model for deeper recommendation versions.
