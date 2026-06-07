# Đặc tả engine quy tắc lý thuyết màu sắc thời trang

## I. Thuật toán chuyển đổi không gian màu từ RGB sang HSL

Dữ liệu màu trang phục ban đầu thường được lưu hoặc trích xuất dưới dạng RGB với $R, G, B \in [0, 255]$. Để đánh giá đặc tính thị giác của con người và tính toán vị trí trên vòng tròn màu, hệ thống chuyển đổi dữ liệu này sang không gian HSL.

### 1. Chuẩn hóa đầu vào

Chuyển ba kênh màu từ thang 8-bit sang hệ số phân số:

$$R' = \frac{R}{255}$$

$$G' = \frac{G}{255}$$

$$B' = \frac{B}{255}$$

Xác định giá trị lớn nhất, nhỏ nhất và độ chênh:

$$C_{max} = \max(R', G', B')$$

$$C_{min} = \min(R', G', B')$$

$$\Delta = C_{max} - C_{min}$$

### 2. Tính độ sáng Lightness ($L$)

Độ sáng là mức trung bình về cảm nhận sáng tối của màu:

$$L = \frac{C_{max} + C_{min}}{2}$$

### 3. Tính độ bão hòa Saturation ($S$)

Độ bão hòa biểu diễn độ tinh khiết của màu:

- **Trường hợp vô sắc ($\Delta = 0$)**  
  $$S = 0$$

- **Trường hợp có sắc độ ($\Delta > 0$)**  
  $$
  S = \begin{cases}
  \frac{\Delta}{C_{max} + C_{min}} & \text{nếu } L \le 0.5 \\
  \frac{\Delta}{2 - (C_{max} + C_{min})} & \text{nếu } L > 0.5
  \end{cases}
  $$

### 4. Tính Hue ($H$)

Hue là vị trí góc của màu trên vòng tròn màu 360 độ:

$$
H = \begin{cases}
0^\circ & \text{nếu } \Delta = 0 \\
60^\circ \times \left( \frac{G' - B'}{\Delta} \bmod 6 \right) & \text{nếu } C_{max} = R' \\
60^\circ \times \left( \frac{B' - R'}{\Delta} + 2 \right) & \text{nếu } C_{max} = G' \\
60^\circ \times \left( \frac{R' - G'}{\Delta} + 4 \right) & \text{nếu } C_{max} = B'
\end{cases}
$$

### Phiên bản hiện tại

Phiên bản hiện tại của hệ thống chưa nên bị hiểu là toàn bộ công thức trên đang được sử dụng như một engine hình học hoàn chỉnh trong mọi luồng.

Tuy nhiên, dữ liệu màu vẫn đang có vai trò thật trong backend hiện tại:

- được AI trích xuất khi xử lý ảnh item
- được lưu thành metadata của item
- được dùng trong rich text context để sinh embedding

Nói cách khác, phần công thức vẫn được giữ như nền tảng lý thuyết mục tiêu, còn vai trò hiện tại của màu sắc là dữ liệu đầu vào quan trọng cho AI và cho các hướng phối đồ nâng cao.

---

## II. Pipeline lọc màu vô sắc

Trước khi chạy các phép ghép cặp theo góc hình học, hệ thống tách các màu trung tính ra khỏi các màu có điểm neo sắc độ rõ ràng.

- Một item được xếp vào `Neutral_Pool` thay vì `Chroma_Pool` nếu thỏa một trong các điều kiện:
  - **Đen:** $L \le 10\%$
  - **Trắng:** $L \ge 90\%$
  - **Xám:** $S \le 10\%$

### Thiết kế mục tiêu

Thiết kế này giúp hệ thống:

- tránh ép các màu trung tính vào các luật hình học không cần thiết
- cho phép item trung tính kết hợp linh hoạt hơn

### Phiên bản hiện tại

Phiên bản hiện tại của code chưa nên được mô tả là đã hiện thực đầy đủ pipeline `Neutral_Pool` hoặc `Chroma_Pool` theo đúng đặc tả cũ ở mọi bước recommendation.

Tuy vậy, khái niệm này vẫn nên được giữ trong docs vì:

- nó là nền lý thuyết hợp lý cho hệ phối đồ
- nó có thể được dùng lại trong local swap, recommendation nâng cao hoặc các bộ lọc cục bộ sau này

---

## III. Quy tắc phối màu hình học

Với các item còn mang màu sắc có cấu trúc rõ, engine sẽ tính độ lệch góc trên vòng tròn màu:

$$\Delta H = |H_{\text{Item1}} - H_{\text{Item2}}|$$

$$\Delta H_{\text{final}} = \min(\Delta H, 360^\circ - \Delta H)$$

### 1. Tổ hợp màu tương đồng

Mục tiêu là tạo các outfit hài hòa và ít tương phản.

- **Ràng buộc chính:**

$$\Delta H_{\text{final}} < 30^\circ$$

- **Ràng buộc tương phản sáng tối tối thiểu:**

$$|L_{\text{Item1}} - L_{\text{Item2}}| \ge 15\%$$

### 2. Tổ hợp màu bổ sung

Mục tiêu là tạo phối màu tương phản mạnh.

- **Ràng buộc chính:**

$$165^\circ \le \Delta H_{\text{final}} \le 195^\circ$$

### Phiên bản hiện tại

Các quy tắc này vẫn được giữ nguyên trong tài liệu như **engine lý thuyết mục tiêu** của hệ thống phối đồ.

Ở trạng thái hiện tại:

- backend đã có AI recommendation
- backend đã có metadata màu
- backend đã có các mô tả trong tài liệu khác về stage lọc màu hoặc style matrix

Nhưng không nên khẳng định quá mức rằng mọi công thức ở đây đang chạy trực tiếp và đầy đủ trong toàn bộ implementation hiện tại nếu code chưa thể hiện rõ từng bước.

---

## IV. Sơ đồ tương tác dữ liệu đầu ra

Sau khi backend hoàn thành việc tính toán các cặp phối màu, payload có thể được đóng gói thành schema chuẩn cho các bước AI hoặc tổng hợp phía sau.

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
          "name": "Áo thun cam",
          "hex": "#FF5733",
          "hsl": [11, 100, 60]
        },
        "bottom": {
          "id": "uuid-2",
          "name": "Quần jean xanh dương",
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

### Định hướng đầu ra mục tiêu theo recommendation hiện tại

Khi đối chiếu với DTO recommendation hiện tại, lớp dữ liệu màu và các cặp đã tiền kiểm có thể được dùng như đầu vào để tạo ra output nghiệp vụ ở dạng:

```json
{
  "title": "Bộ casual đi chơi cuối tuần",
  "explanation": "Phối hợp tông trung tính với điểm nhấn màu nhẹ, phù hợp thời tiết và phong cách người dùng",
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

Schema này cho thấy dữ liệu lý thuyết màu không nhất thiết phải đi thẳng ra ngoài dưới dạng cặp màu thuần túy. Thay vào đó, nó có thể được tiêu hóa thành:

- tiêu đề outfit
- giải thích lựa chọn
- các nhóm item theo vai trò
- item chính và item thay thế cho từng vai trò

### Phiên bản hiện tại

Schema trên vẫn là mô tả rất hữu ích của **dữ liệu mục tiêu** cho các bước tổng hợp outfit bằng AI hoặc các bộ lọc cục bộ.

Ở backend hiện tại:

- item đã có dữ liệu màu và embedding
- rich text context đã sử dụng metadata màu
- recommendation engine đã có nền dữ liệu để tiến dần tới dạng payload như trên

Do đó, phần này không bị loại bỏ mà được hiểu là mô hình dữ liệu đích cho các phiên bản recommendation sâu hơn.
