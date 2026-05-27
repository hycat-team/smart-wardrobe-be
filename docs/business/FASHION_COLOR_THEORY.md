# FASHION COLOR THEORY RULE ENGINE SPECIFICATION

## I. COLOR SPACE CONVERSION ALGORITHM (RGB TO HSL)

Raw apparel color data is initially stored or extracted as RGB components where $R, G, B \in [0, 255]$. To evaluate human visual characteristics and calculate geometric positions on a color wheel, the system transforms RGB coordinates into the HSL (Hue, Saturation, Lightness) color space.

### 1. Input Normalization

Convert raw 8-bit integer color channels into fractional coefficients:

$$R' = \frac{R}{255}$$

$$G' = \frac{G}{255}$$

$$B' = \frac{B}{255}$$

Identify boundary extremes and their variance interval:

$$C_{max} = \max(R', G', B')$$

$$C_{min} = \min(R', G', B')$$

$$\Delta = C_{max} - C_{min}$$

### 2. Lightness ($L$) Computation

Lightness represents the average luminance value bounded between $0.0$ (pure black) and $1.0$ (pure white):

$$L = \frac{C_{max} + C_{min}}{2}$$

### 3. Saturation ($S$) Computation

Saturation defines the purity or intensity of the color hue, scaling from $0.0$ (completely grayscale) to $1.0$ (maximum intensity). The math formula branches conditionally based on the calculated $L$ value:

- **Case 1: Achromatic Grayscale ($\Delta = 0$)**

$$S = 0$$

- **Case 2: Chromatic Color ($\Delta > 0$)**
    $$
    S = \begin{cases}
    \frac{\Delta}{C_{max} + C_{min}} & \text{if } L \le 0.5 \ \
    \frac{\Delta}{2 - (C_{max} + C_{min})} & \text{if } L > 0.5
    \end{cases}
    $$

### 4. Hue ($H$) Computation

Hue establishes the absolute radial degree location on the 360-degree color wheel wheel profile:

$$
H = \begin{cases}
0^\circ & \text{if } \Delta = 0 \
60^\circ \times \left( \frac{G' - B'}{\Delta} \bmod 6 \right) & \text{if } C_{max} = R' \
60^\circ \times \left( \frac{B' - R'}{\Delta} + 2 \right) & \text{if } C_{max} = G' \
60^\circ \times \left( \frac{R' - G'}{\Delta} + 4 \right) & \text{if } C_{max} = B'
\end{cases}
$$

---

## II. ACHROMATIC FILTERING PIPELINE

Before running geometric angle pairings, the system filters out neutral colors lacking a fixed chromatic anchor point. These items can safely pair with any other shade.

- **Rule Validation Filter:** An apparel item is classified into the `Neutral_Pool` instead of the `Chroma_Pool` if it meets any of the following boundary metrics:

- **Black Isolation:** Lightness $L \le 10\%$

- **White Isolation:** Lightness $L \ge 90\%$

- **Gray Isolation:** Saturation $S \le 10\%$

---

## III. GEOMETRIC COLOR COHESION PAIRING

For clothing items retaining structural colors (`Chroma_Pool`), the rule engine executes a combinatorial permutation iteration. It loops through distinct items across contrasting categories to compute absolute radial angular deviation over the color wheel boundaries:

$$\Delta H = |H_{\text{Item1}} - H_{\text{Item2}}|$$

$$\Delta H_{\text{final}} = \min(\Delta H, 360^\circ - \Delta H)$$

### 1. Analogous Styling Combination

This routine isolates adjacent color configurations on the wheel layout to construct harmonious, low-contrast ensembles.

- **Primary Constraint:**

$$\Delta H_{\text{final}} < 30^\circ$$

- **Visual Contrast Adjustment Rule:** To prevent flat, monotonal looks and inject depth, the items must display a minimum luminance separation:

$$|L_{\text{Item1}} - L_{\text{Item2}}| \ge 15\%$$

### 2. Complementary Styling Combination

This routine pairs polar opposite color nodes across the wheel diameter to formulate high-impact, striking contrast palettes.

- **Primary Constraint (Allowing a $15^\circ$ Margin of Error):**

$$165^\circ \le \Delta H_{\text{final}} \le 195^\circ$$

---

## IV. DATA PAYLOAD INTERACTION SCHEME

Once the backend RAM matrix finishes calculating these combinatorial pairs, it packages the payload into a structured schema ready for downstream LLM analysis (For example, please make changes if necessary.):

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
                    "name": "Áo thun Cam",
                    "hex": "#FF5733",
                    "hsl": [11, 100, 60]
                },
                "bottom": {
                    "id": "uuid-2",
                    "name": "Quần jean Xanh dương",
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
