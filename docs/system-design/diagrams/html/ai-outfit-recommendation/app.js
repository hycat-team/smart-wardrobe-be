const TRANSLATIONS = {
    vi: {
        theme: "Giao diện",
        interactivePipeline: "Sơ đồ tương tác",
        overview: "Tổng quan",
        welcomeTitle: "Interactive Pipeline",
        welcomeBody: `Sơ đồ tương tác này trực quan hóa quy trình gợi ý trang phục Multi-stage RAG. Hãy chọn một kịch bản bằng các nút phía trên để xem luồng thực thi chạy thử, hoặc click trực tiếp vào các component trên sơ đồ để khám phá chi tiết thuật toán, dữ liệu vào/ra và các file mã nguồn Go tương ứng.`,
        barTitle: "AI Outfit Recommendation Pipeline",
        barSub: "Multi-stage RAG Algorithm &amp; Local Fallback Flow",
        sourceLabel: "Closy System Design · internal/modules/fashion/application/usecase/ai/recommendation",
        flowcapName: "Các Bước Thực Thi",
        chips: {
            all: "Tổng quan",
            happy: "Luồng chuẩn (LLM RAG)",
            fallback: "Luồng dự phòng HSL",
            padding: "Mở rộng ứng viên",
            hybrid: "Chi tiết Tìm kiếm lai"
        },
        DETAIL: {
            request: {
                t: "User Outfit Request",
                m: "Trigger Node · Client Input & Pre-checks",
                b: "API entrypoint: <code>POST /v1/ai/recommendations</code>.<br>Hệ thống bắt đầu bằng việc chạy các bộ lọc tiền xử lý và kiểm soát điều kiện biên bảo vệ hệ thống:<br>• <b>Kiểm tra Quota:</b> Xác thực và reset hạn ngạch sử dụng trong ngày của người dùng qua <code>GetAndResetDailyQuota</code>.<br>• <b>Lọc đồ hoạt động:</b> Lấy danh sách đồ có trạng thái <code>InWardrobe</code> và chưa bị xóa logic.<br>• <b>Kiểm soát số lượng tối thiểu:</b> Tủ đồ hoạt động bắt buộc phải có ít nhất <b>5 sản phẩm</b>; nếu không thỏa mãn, hệ thống chặn xử lý và trả về lỗi <code>ErrMinimumWardrobeItemsRequired</code>.",
            },
            nlpparse: {
                t: "nlpParser.Parse",
                m: "NLP Intent Extraction · internal/modules/ai/nlp",
                b: "Bộ phân tích cú pháp NLP xử lý chuỗi văn bản tự do của người dùng qua 6 cấu phần tuần tự chạy ngầm trong RAM:<br>1. <b>NormalizeText:</b> Chuẩn hóa Telex mồ côi, đệm khoảng trắng dấu câu, và mã hóa chặn đầu phủ định (như <i>'đừng'</i> thành <code>'dung_neg'</code>) để tránh mất dấu logic phủ định khi lột dấu.<br>2. <b>splitSentences:</b> Chia cắt câu thành các mệnh đề độc lập dựa trên liên từ tương phản và dấu câu, tránh hiện tượng lan truyền phủ định sai lệch (Negation Scope Creep).<br>3. <b>detectMatches:</b> Khớp từ điển tĩnh (Dịp, Phong cách, Màu, Thời tiết) dùng Pre-compiled Regex tối ưu CPU và bẫy ranh giới từ <code>\\b</code>. Ghi nhận vị trí byte Start/End và sắp xếp tuyến tính.<br>4. <b>isNegatedMatch:</b> Sử dụng cửa sổ trượt ngược (Sliding Window) tối đa 4 từ phía trước thực thể. Nếu gặp từ phủ định (như <code>'khong'</code>, <code>'dung_neg'</code>...), thực thể bị đột biến sang trạng thái loại trừ (như <code>ExcludedStyles</code> hoặc <code>ExcludedColorTones</code>).<br>5. <b>detectRawLexicalTerms:</b> Thu gom từ thô ngoài từ điển (FTS keywords) có độ dài > 2 ký tự, không trùng từ dừng (Stopwords), và chạy hàm <code>overlapsAnyMatch</code> để loại bỏ các token đè lên thực thể từ điển đã khớp.<br>6. <b>RemoveConflictingLexicalTerms:</b> Khử mâu thuẫn logic giữa bộ lọc cứng trên UI (ví dụ: Dịp = Work) và từ khóa thô trong văn bản (ví dụ: 'đi chơi' -> casual) để tránh SQL trả về tập hợp rỗng.",
            },
            buildquery: {
                t: "buildRecommendationRetrievalQuery",
                m: "Query Rewriting & Decision · candidate_helpers.go",
                b: "Dựng cấu trúc truy vấn tìm kiếm ứng viên <code>types.RecommendationRetrievalQuery</code>. Logic hoạt động:<br>• Hệ thống kiểm tra cờ cấu hình <code>RecommendationLLMRewriterEnabled</code>.<br>• <b>Nếu bật (True):</b> Gọi dịch vụ LLM (<code>LLMRecommendationQueryRewriter.Rewrite</code>) để viết lại câu truy vấn bằng AI một cách tự nhiên.<br>• <b>Nếu tắt hoặc gặp lỗi (False / Error):</b> Hệ thống tự động fallback sang cơ chế <code>LocalRecommendationQueryRewriter.Rewrite</code> để mở rộng truy vấn tĩnh bằng rule-based:<br>&nbsp;&nbsp;1. <b>BuildSourceAwareRetrievalTerms:</b> Ánh xạ thuộc tính ý định rõ ràng (Dịp, Phong cách, Tông màu, Ràng buộc tích cực) -> nguồn <code>dictionary</code>, và từ khóa tự do -> nguồn <code>raw</code>.<br>&nbsp;&nbsp;2. <b>Mở rộng Taxonomy:</b> Chạy <code>ExpandRecommendationLexicalRetrievalTerms</code> để mở rộng từ khóa theo nhóm từ điển cấu hình sẵn (ví dụ: dịp <code>work</code> thêm <code>office</code>, <code>formal</code>).<br>&nbsp;&nbsp;3. <b>BuildSourceAwareExcludedTerms:</b> Thu thập từ tránh/cấm từ UI và bộ lọc (<code>avoid-style:</code>, <code>avoid-color-tone:</code>...) -> <code>dictionary</code>, mở rộng qua taxonomy cấm <code>taxonomyGroupExcluded</code>.<br>&nbsp;&nbsp;4. <b>NormalizeRetrievalTerms:</b> Chuẩn hóa chữ thường, lọc các từ dừng (<code>LexicalStopwords</code>), và khử trùng lặp có độ ưu tiên: <code>Dictionary (0) > Raw (1) > Taxonomy (2)</code>.<br>&nbsp;&nbsp;5. <b>Hard Filters:</b> Xây dựng bộ lọc cứng <code>Seasonality</code> dựa trên các ràng buộc mùa vụ.",
            },
            generateembeddings: {
                t: "aiService.GenerateEmbeddings",
                m: "Query Vectorization · Shared AI Service",
                b: "Sử dụng dịch vụ AI dùng chung (Shared AI Service) hoạt động độc lập với provider (hỗ trợ cấu hình primary/fallback động giữa Google/OpenAI). Vector hóa câu truy vấn mô tả đã được tối ưu thành một mảng vector 768 chiều để phục vụ cho thuật toán tìm kiếm tương đồng ngữ nghĩa (Cosine Similarity) dưới cơ sở dữ liệu.",
            },
            getcandidates: {
                t: "GetHybridCandidates",
                m: "Hybrid Retrieval & Brand Integration · fashionItemRepo",
                b: "Thực hiện truy vấn ứng viên song song từ database kết hợp hai luồng:<br>1. <b>Vector Search (pgvector + HNSW Index):</b> Tìm kiếm tương đồng ngữ nghĩa. Cơ sở dữ liệu sử dụng chỉ mục <b>HNSW (Hierarchical Navigable Small World)</b> phân tầng: Tầng 0 chứa 100% vector dữ liệu, các tầng trên thưa thớt hơn giúp <i>Greedy Routing</i> nhảy cóc khoảng cách nhanh chóng từ trên xuống dưới. Tại Tầng 0, thuật toán duyệt qua một danh sách ứng viên động kích thước <code>ef_search</code> để tránh cực tiểu cục bộ trước khi trả về Top K kết quả tương đồng Cosine (toán tử <code>&lt;=&gt;</code>). Số cạnh mỗi nút được giới hạn bởi tham số <code>M</code>.<br>2. <b>Lexical Search (Full-Text Search + GIN Index):</b> So khớp từ khóa thô sử dụng chỉ mục đảo <b>GIN (Generalized Inverted Index)</b> trên cột <code>document_tsvector</code> (chứa các từ tố Lexemes không dấu kèm positional offsets). Điểm số được xếp hạng bằng hàm <b><code>ts_rank_cd</code> (Cover Density)</b> đo lường khoảng cách hình học ngắn nhất giữa các từ khóa khớp nhau trong câu (cụm từ đứng sát nhau được điểm cao hơn nhiều so với đứng xa nhau).<br>• Kết quả hai luồng được hợp nhất bằng thuật toán <b>RRF (Reciprocal Rank Fusion)</b>.<br>• <b>Tích hợp Brand Items:</b> Nếu <code>IncludeBrandItems</code> bật, nạp thêm đề xuất bán thêm từ Brand Module, khống chế số lượng <b>tối đa 30%</b> kích thước bể ứng viên tối thiểu (<code>minimumPool</code>) để tránh lấn át tủ đồ cá nhân.",
            },
            ensurepool: {
                t: "EnsureMinimumCandidatePool",
                m: "Belly Expansion Safeguard · pool_expansion.go",
                b: "Thuật toán mở rộng bể ứng viên dự phòng nếu số lượng ứng viên thu được từ tìm kiếm lai nhỏ hơn <code>minimumPool</code>. Hệ thống duyệt qua tủ đồ người dùng và nạp thêm đồ theo 3 cấp độ nới lỏng tăng dần:<br>• <b>Cấp 1 (Strict):</b> Phải khớp bộ lọc cứng (mùa, category) và chứa từ khóa tìm kiếm (<code>MatchesFallbackLexicalTerms</code>).<br>• <b>Cấp 2 (Relaxed):</b> Chỉ cần thỏa mãn bộ lọc cứng về mùa và danh mục sản phẩm.<br>• <b>Cấp 3 (General):</b> Nới lỏng hoàn toàn điều kiện lọc, lấy bất kỳ quần áo nào đang hoạt động.<br><i>Lưu ý:</i> Tất cả các cấp đều loại trừ đồ chứa từ khóa cấm (<code>MatchesFallbackExcludedTerms</code>).",
            },
            candidatepool: {
                t: "Candidate Pool",
                m: "rankingCandidates · Memory Slice",
                b: "Lát cắt danh sách ứng viên (slice của <code>CandidateForRanking</code>) được lưu trong RAM. Đây là dữ liệu ngữ cảnh duy nhất cung cấp cho các bước xếp hạng lại và tạo Prompt, chứa cả thông tin nguồn gốc (<code>UserWardrobe</code> hoặc <code>BrandItem</code>).",
            },
            rankcandidates: {
                t: "RankCandidates / ScoreCandidateItem",
                m: "Heuristic Scorer · ranking/ranking.go",
                b: "Chấm điểm từng món đồ dựa trên công thức heuristic với điểm cơ sở khởi tạo là 1.0:<br>• <b>Style:</b> +0.3 nếu khớp style target; -0.35 nếu là style cần tránh.<br>• <b>Occasion:</b> +0.2 nếu khớp trực tiếp; +0.08 nếu khớp qua Taxonomy.<br>• <b>Color Tone:</b> +0.2 nếu đúng tông sáng/tối/đất; -0.25 nếu là tông cần tránh; +0.1 cho màu trung tính dễ phối.<br>• <b>Thời tiết:</b> Áo khoác +0.4 khi trời lạnh, -0.3 khi trời nóng; Đồ đi mưa +0.15 khi trời mưa. Phạt -0.2 đến -0.3 nếu trùng thời tiết cần tránh.<br>• <b>Lexical Boost:</b> Cộng tối đa +0.5 dựa trên độ trùng khớp từ khóa thô.<br>• <b>Lịch sử dùng:</b> Phạt tối đa -0.4 nếu mới mặc gần đây (giảm tuyến tính); Thưởng +0.15 nếu lâu chưa mặc hoặc đồ mới.<br>• <b>Retrieval Bonus:</b> Thêm tối đa +0.25 dựa trên điểm và thứ hạng tìm kiếm lai.<br>• <b>Fallback Penalty:</b> Trừ điểm dựa trên độ nới lỏng nguồn ứng viên (Strict: -0.1, Relaxed: -0.2, General: -0.35).",
            },
            diversify: {
                t: "DiversifyRankedCandidates",
                m: "Soft Cap Diversity Filter · ranking.go",
                b: "Đảm bảo tính đa dạng của kết quả gợi ý gửi cho LLM. Thuật toán hoạt động qua 2 vòng:<br>1. <b>Vòng 1 (Soft Cap):</b> Duyệt danh sách đã sắp xếp điểm. Chỉ lấy tối đa <b>3 sản phẩm</b> cho mỗi danh mục (áo, quần, váy...). Các sản phẩm dư thừa bị chuyển vào danh sách trì hoãn (<code>deferred</code>).<br>2. <b>Vòng 2 (Lấp đầy):</b> Nếu bể chọn lọc chưa đạt tới giới hạn pool, tiếp tục rút các món đồ từ danh sách hoãn lại đưa vào cho đến khi đủ số lượng.",
            },
            buildprompt: {
                t: "BuildRecommendationPromptWithLimits",
                m: "Dynamic Prompt Assembly · synthesis/prompt.go",
                b: "Tạo prompt gửi cho LLM chứa các quy luật phối đồ chặt chẽ (như đầm không đi kèm áo/quần, áo khoác mặc ngoài...) và ngữ cảnh người dùng.<br>• <b>Mã hóa Alias:</b> ID của sản phẩm được rút gọn thành <code>A1</code>, <code>A2</code>, <code>A3</code>... giúp giảm lượng token tiêu thụ và triệt tiêu hoàn toàn lỗi LLM tự chế (ảo giác) UUID.<br>• <b>Khống chế kích thước động:</b> Nếu tổng chiều dài prompt vượt quá <code>PromptMaxCharacters</code>, thuật toán sẽ chủ động loại bỏ dần từng ứng viên ở cuối danh sách xếp hạng cho đến khi prompt ngắn hơn giới hạn an toàn.",
            },
            generatetext: {
                t: "GenerateRecommendationText",
                m: "LLM Styling Synthesis · Shared AI Service",
                b: "Gửi prompt tới mô hình ngôn ngữ lớn (Gemini hoặc OpenAI tùy cấu hình fallback). Thiết lập các tham số tối ưu:<br>• <b>Nhiệt độ (Temperature) = 0.1:</b> Ràng buộc mô hình hoạt động ổn định, tránh sự sáng tạo quá đà gây vi phạm quy tắc phối đồ.<br>• <b>JSON Schema:</b> Sử dụng tính năng ép kiểu MIME <code>application/json</code> kết hợp truyền schema định nghĩa chặt chẽ để đảm bảo kết quả trả về luôn khớp cấu trúc mong muốn.",
            },
            isllmsuccess: {
                t: "LLM Success Check",
                m: "Resilience Decision Diamond",
                b: "Điểm rẽ nhánh quan trọng của hệ thống:<br>• <b>Nhánh Success:</b> Nếu LLM trả về chuỗi JSON hợp lệ, chuyển tiếp tới bộ giải mã và xác thực.<br>• <b>Nhánh Failure:</b> Nếu LLM gặp lỗi (hết quota, timeout, lỗi mạng, safety block), hệ thống gọi hàm <code>ClassifyFallbackTrace</code> để phân loại lỗi lưu log telemetry, đồng thời kích hoạt động cơ phối đồ dự phòng cục bộ (local fallback) để đảm bảo API luôn hoạt động (100% Availability).",
            },
            resolution: {
                t: "Response Validation & ID Resolution",
                m: "Sanitization & Alias Decoding · mapper/recommendation.go",
                b: "Bộ xử lý và làm sạch kết quả đầu ra của LLM:<br>1. <b>Làm sạch JSON:</b> Xóa các tag markdown code block và dùng hàm <code>ExtractFirstJSONObject</code> để cắt ra đối tượng JSON hợp lệ đầu tiên.<br>2. <b>Giải mã Alias:</b> Chuyển ngược các nhãn <code>A1</code>, <code>A2</code>... về lại UUID gốc trong database.<br>3. <b>Lọc Placeholder:</b> Từ chối kết quả nếu chứa các giá trị giả định của AI (như <i>'string'</i>, <i>'uuid'</i>, <i>'role'</i>) qua <code>ValidateOutfitRecommendationPayload</code>.<br>4. <b>Map DTO:</b> Ánh xạ sản phẩm thực tế, tự động loại bỏ các cặp áo/quần đề xuất nếu bộ phối đã chọn một đầm liền thân (fullbody).",
            },
            hslmatching: {
                t: "RunLocalHSLMatching",
                m: "Local Heuristic HSL Color Harmony · fallback_hsl.go",
                b: "Thuật toán phối đồ dự phòng cục bộ không phụ thuộc AI, đảm bảo thời gian hoạt động liên tục (100% Availability) của hệ thống:<br>• <b>1. Phân nhóm danh mục ứng viên:</b> Chia nhỏ các món đồ từ bể ứng viên đã sắp xếp điểm (Candidates Pool) thành các danh sách độc lập: <i>Áo (tops), Quần/Chân váy (bottoms), Giày (shoes), Áo khoác (outerwears), Đầm (dresses) và Phụ kiện (accessories)</i> dựa trên thuộc tính <code>CategorySlug</code>.<br>• <b>2. Nhận dạng điều kiện thời tiết (Weather Parsing):</b> Phân tích thuộc tính thời tiết đầu vào (chuyển sang chữ thường). Nếu chứa các từ khóa như <code>'lạnh', 'mát', 'cold', 'cool', 'mưa', 'rain'</code>, hệ thống thiết lập cờ <code>isCold = true</code> để kích hoạt logic nạp thêm áo khoác ngoài.<br>• <b>3. Thuật toán ghép bộ chính:</b><br>&nbsp;&nbsp;- <i>Trường hợp Đầm & Giày:</i> Nếu có đầm và giày (nhưng thiếu áo hoặc quần), lấy đầm đầu tiên làm trung tâm. Duyệt tìm đôi giày đầu tiên hợp màu với đầm. Nếu <code>isCold = true</code>, tiếp tục tìm chiếc áo khoác ngoài đầu tiên hợp màu với đầm.<br>&nbsp;&nbsp;- <i>Trường hợp Áo & Quần/Chân váy:</i> Duyệt tích cực qua tất cả các cặp áo-quần để tìm <b>cặp phối màu hài hòa đầu tiên</b> làm đồ chính. Sau đó, tìm đôi giày và chiếc áo khoác ngoài (nếu lạnh) hợp màu với <b>áo hoặc quần chính</b>.<br>&nbsp;&nbsp;- <i>Mặc định Fallback:</i> Nếu hoàn toàn không tìm thấy cặp nào hợp màu, hệ thống tự động chọn món đồ đầu tiên của mỗi nhóm (Áo, Quần, Giày) để ghép bộ.<br>• <b>4. Quy tắc phối màu HSL (colorsMatch):</b> Hai món đồ được coi là hợp màu khi thỏa mãn một trong các điều kiện:<br>&nbsp;&nbsp;1. <i>Màu trung tính:</i> Độ sáng L &le; 10.0 (đen), L &ge; 90.0 (trắng) hoặc độ bão hòa S &le; 10.0 (xám/màu nhạt). Màu trung tính luôn được coi là hợp màu với mọi màu sắc khác.<br>&nbsp;&nbsp;2. <i>Màu tương đồng (Analogous):</i> Khoảng cách góc màu &Delta;H = min(|H<sub>1</sub> - H<sub>2</sub>|, 360 - |H<sub>1</sub> - H<sub>2</sub>|) &lt; 30&deg; và chênh lệch độ sáng |L<sub>1</sub> - L<sub>2</sub>| &ge; 15.0 để tránh hiện tượng phối đồ bị chìm màu.<br>&nbsp;&nbsp;3. <i>Màu đối lập (Complementary):</i> Góc màu nằm ở phía đối diện của bánh xe màu sắc: &Delta;H &isin; [165.0, 195.0].<br>• <b>5. Nhóm đồ thay thế (Alternatives):</b> Mỗi vị trí trong bộ đồ đề xuất (ví dụ: Áo chính) sẽ được hệ thống đính kèm tối đa <b>2 món đồ thay thế</b> từ cùng danh mục trong pool để người dùng tùy ý chuyển đổi trên giao diện.",
            },
            dtores: {
                t: "dto.RecommendedOutfitRes",
                m: "Final Output DTO",
                b: "Cấu trúc phản hồi cuối cùng trả về cho client gồm tiêu đề, giải thích phối đồ bằng tiếng Việt có dấu, mảng các nhóm đồ phối (đồ chính + đồ thay thế) kèm theo cờ <code>IsFallback</code> để client biết và hiển thị biểu tượng chỉ báo chế độ dự phòng tương ứng trên giao diện người dùng.",
            },
        },
        FLOWS: {
            happy: {
                name: "Luồng chuẩn — Standard LLM RAG",
                steps: [
                    "Client gửi yêu cầu gợi ý chứa sở thích, thời tiết, và style mục tiêu.",
                    "<code>nlpParser.Parse</code> trích xuất các ràng buộc ngữ nghĩa từ prompt đầu vào.",
                    "<code>buildRetrievalQuery</code> viết lại câu truy vấn cho cả tìm kiếm Vector và Lexical.",
                    "<code>aiService.GenerateEmbeddings</code> vector hóa các tiêu chí tìm kiếm.",
                    "<code>GetHybridCandidates</code> thực hiện tìm kiếm song song, kết hợp kết quả qua RRF.",
                    "Ứng viên được lưu trữ trong <code>Candidate Pool</code>.",
                    "<code>RankCandidates</code> tính điểm dựa trên độ khớp style, thời tiết, lịch sử mặc, và nguồn.",
                    "<code>DiversifyCandidates</code> lọc kết quả để đảm bảo phân bổ danh mục cân bằng (Soft Cap).",
                    "<code>BuildRecomPrompt</code> dựng prompt có cấu trúc chứa mã định danh (ví dụ <code>A1</code>, <code>A2</code>).",
                    "<code>GenerateRecText</code> gửi prompt tới LLM (Gemini) để phối đồ.",
                    "LLM phản hồi thành công, trả về các bộ phối dạng JSON có cấu trúc.",
                    "<code>Response Validation</code> parse JSON và giải mã ID từ alias thành UUID thật.",
                    "Gợi ý trang phục hoàn chỉnh được trả về trong DTO kết quả."
                ]
            },
            fallback: {
                name: "Luồng dự phòng — Local HSL",
                steps: [
                    "Client gửi yêu cầu gợi ý chứa sở thích, thời tiết, và style mục tiêu.",
                    "<code>nlpParser.Parse</code> trích xuất các ràng buộc ngữ nghĩa.",
                    "<code>buildRetrievalQuery</code> viết lại câu truy vấn.",
                    "<code>aiService.GenerateEmbeddings</code> vector hóa câu truy vấn.",
                    "<code>GetHybridCandidates</code> tìm kiếm ứng viên kết hợp Vector + FTS.",
                    "Ứng viên được nạp vào <code>Candidate Pool</code>.",
                    "<code>RankCandidates</code> chấm điểm chi tiết các ứng viên.",
                    "<code>DiversifyCandidates</code> lọc đa dạng danh mục và giữ các món điểm cao nhất.",
                    "<code>BuildRecomPrompt</code> dựng prompt và gửi tới LLM.",
                    "LLM gặp lỗi (hết quota, timeout hoặc block an toàn).",
                    "Nhánh rẽ nhánh xác nhận lỗi LLM và kích hoạt chế độ dự phòng.",
                    "<code>RunLocalHSLMatching</code> chạy thuật toán phối màu HSL cục bộ lập tức.",
                    "Gợi ý trang phục dự phòng kèm cờ <code>IsFallback = true</code> được gửi về client."
                ]
            },
            padding: {
                name: "Luồng mở rộng ứng viên dự bị",
                steps: [
                    "Khởi động tìm kiếm ứng viên qua Vector và Lexical Search.",
                    "Đếm tổng số lượng ứng viên thu được. Nếu số lượng nhỏ hơn giới hạn tối thiểu.",
                    "Kích hoạt hàm kiểm tra <code>EnsureMinimumCandidatePool</code>.",
                    "Quét tủ đồ ở **Cấp 1 (Strict)**: Lọc nghiêm ngặt theo mùa và từ khóa tích cực.",
                    "Nếu vẫn thiếu, quét ở **Cấp 2 (Relaxed)**: Chỉ lọc theo mùa và danh mục.",
                    "Nếu vẫn chưa đủ, quét ở **Cấp 3 (General)**: Lấy mọi quần áo hoạt động.",
                    "Hợp nhất và loại bỏ các món trùng lặp để lấp đầy <code>Candidate Pool</code>."
                ]
            },
            hybrid: {
                name: "Chi tiết quy trình Tìm kiếm lai",
                steps: [
                    "Nhận từ khóa truy vấn thô và vector hóa tương ứng từ bước trước.",
                    "**Nhánh Vector:** pgvector tính khoảng cách Cosine trên chỉ mục HNSW.",
                    "HNSW di chuyển qua các tầng đồ thị để tìm nhanh lân cận gần nhất.",
                    "**Nhánh Lexical:** GIN tìm kiếm đảo các từ tố khớp trong <code>tsvector</code>.",
                    "Sử dụng hàm <code>ts_rank_cd</code> chấm điểm dựa trên mật độ khoảng cách từ khóa.",
                    "Kết quả từ 2 nhánh được chuẩn hóa và xếp hạng lại bằng thuật toán RRF.",
                    "Các ứng viên có điểm RRF cao nhất được nạp vào Candidate Pool."
                ]
            }
        }
    },
    en: {
        theme: "Theme",
        interactivePipeline: "Interactive Diagram",
        overview: "Overview",
        welcomeTitle: "Interactive Pipeline",
        welcomeBody: `This interactive diagram visualizes the Multi-stage RAG recommendation pipeline. Select a scenario using the chips above to highlight specific execution flows, or click individual pipeline stages to explore their functional details, inputs, outputs, and exact Go codebase implementations.`,
        barTitle: "AI Outfit Recommendation Pipeline",
        barSub: "Multi-stage RAG Algorithm &amp; Local Fallback Flow",
        sourceLabel: "Closy System Design · internal/modules/fashion/application/usecase/ai/recommendation",
        flowcapName: "Execution Steps",
        chips: {
            all: "Overview",
            happy: "Happy Path (LLM RAG)",
            fallback: "HSL Fallback Path",
            padding: "Candidate Expansion",
            hybrid: "Hybrid Search Detail"
        },
        DETAIL: {
            request: {
                t: "User Outfit Request",
                m: "Trigger Node · Client Input & Pre-checks",
                b: "API entrypoint: <code>POST /v1/ai/recommendations</code>.<br>The system starts by executing preprocessing filters and edge cases guarding system integrity:<br>• <b>Quota Check:</b> Validates and resets the user's daily usage quota using <code>GetAndResetDailyQuota</code>.<br>• <b>Active Items Filter:</b> Fetches fashion items with status <code>InWardrobe</code> that are not logically deleted.<br>• <b>Minimum Count Guard:</b> The wardrobe must contain at least <b>5 active items</b>; otherwise, the system blocks execution and throws error <code>ErrMinimumWardrobeItemsRequired</code>.",
            },
            nlpparse: {
                t: "nlpParser.Parse",
                m: "NLP Intent Extraction · internal/modules/ai/nlp",
                b: "The NLP parser processes the user's free-text input through 6 sequential RAM-bound components:<br>1. <b>NormalizeText:</b> Standardizes lone Telex markers, pads punctuation spacing, and prefix-encodes negations (e.g., converting <i>'don\\'t'</i> to <code>'dung_neg'</code>) to prevent losing negative logic during accent stripping.<br>2. <b>splitSentences:</b> Splits sentences into independent clauses based on contrastive conjunctions and punctuation to prevent Negation Scope Creep.<br>3. <b>detectMatches:</b> Performs static dictionary matches (Occasion, Style, Color, Weather) using CPU-optimized Pre-compiled Regexes with word boundaries <code>\\b</code>. Byte offsets are sorted linearly.<br>4. <b>isNegatedMatch:</b> Inspects a backward sliding window of up to 4 words prior to the matched entity. If negation tokens (e.g. <code>'khong'</code>, <code>'dung_neg'</code>...) are found, the entity mutates to an exclusion constraint (e.g., <code>ExcludedStyles</code> or <code>ExcludedColorTones</code>).<br>5. <b>detectRawLexicalTerms:</b> Gathers out-of-dictionary raw terms (FTS keywords) longer than 2 characters, skipping stopwords, and runs <code>overlapsAnyMatch</code> to prune tokens overlapping pre-matched dictionary entities.<br>6. <b>RemoveConflictingLexicalTerms:</b> Resolves logic conflicts between hard filters on the UI (e.g. Occasion = Work) and raw textual keywords (e.g., 'hangout' -> casual) to avoid empty SQL results.",
            },
            buildquery: {
                t: "buildRecommendationRetrievalQuery",
                m: "Query Rewriting & Decision · candidate_helpers.go",
                b: "Constructs the candidate search query structure <code>types.RecommendationRetrievalQuery</code>. Pipeline logic:<br>• The system checks configuration flag <code>RecommendationLLMRewriterEnabled</code>.<br>• <b>If Enabled (True):</b> Calls the LLM service (<code>LLMRecommendationQueryRewriter.Rewrite</code>) to naturally rewrite the query using AI.<br>• <b>If Disabled or Errors (False / Error):</b> Automatically falls back to <code>LocalRecommendationQueryRewriter.Rewrite</code> for rule-based query expansion:<br>&nbsp;&nbsp;1. <b>BuildSourceAwareRetrievalTerms:</b> Maps clear intent parameters (Occasions, Styles, Color Tones, positive constraints) -> source <code>dictionary</code>, and free keywords -> source <code>raw</code>.<br>&nbsp;&nbsp;2. <b>Taxonomy Expansion:</b> Runs <code>ExpandRecommendationLexicalRetrievalTerms</code> to expand keywords using predefined semantic dictionary groups (e.g. occasion <code>work</code> adds <code>office</code>, <code>formal</code>).<br>&nbsp;&nbsp;3. <b>BuildSourceAwareExcludedTerms:</b> Gathers avoided/banned properties from UI and filters (<code>avoid-style:</code>, <code>avoid-color-tone:</code>...) -> <code>dictionary</code>, expanded via <code>taxonomyGroupExcluded</code>.<br>&nbsp;&nbsp;4. <b>NormalizeRetrievalTerms:</b> Lowercases, filters out <code>LexicalStopwords</code>, and deduplicates with priority order: <code>Dictionary (0) > Raw (1) > Taxonomy (2)</code>.<br>&nbsp;&nbsp;5. <b>Hard Filters:</b> Generates the <code>Seasonality</code> hard filter constraints based on seasonal properties.",
            },
            generateembeddings: {
                t: "aiService.GenerateEmbeddings",
                m: "Query Vectorization · Shared AI Service",
                b: "Invokes the shared AI service (operating provider-agnostically with dynamic primary/fallback switching between Google and OpenAI). Vectorizes the optimized search query text into a 768-dimensional float array to support semantic similarity search (Cosine Similarity) inside the database.",
            },
            getcandidates: {
                t: "GetHybridCandidates",
                m: "Hybrid Retrieval & Brand Integration · fashionItemRepo",
                b: "Queries candidate items in parallel from PostgreSQL combining two retrieval paths:<br>1. <b>Vector Search (pgvector + HNSW Index):</b> Finds semantic similarity matches. The DB utilizes a hierarchical <b>HNSW (Hierarchical Navigable Small World)</b> index: Level 0 holds 100% of vectors, while upper layers are progressively sparser, allowing <i>Greedy Routing</i> to skip distances rapidly. At Level 0, the search crawls a dynamic candidate list of size <code>ef_search</code> to bypass local minima before returning Top K Cosine similarity results (using operator <code>&lt;=&gt;</code>). Node links are capped by <code>M</code>.<br>2. <b>Lexical Search (Full-Text Search + GIN Index):</b> Matches raw text tokens using a <b>GIN (Generalized Inverted Index)</b> index over the <code>document_tsvector</code> column (containing tokenized accent-stripped lexemes and positional offsets). Score is ranked using <b><code>ts_rank_cd</code> (Cover Density)</b> which measures the shortest geometric distance between matching lexemes (adjacent terms score much higher than dispersed terms).<br>• Results from both pipelines are merged using <b>RRF (Reciprocal Rank Fusion)</b>.<br>• <b>Brand Items Integration:</b> If <code>IncludeBrandItems</code> is enabled, loads commercial recommendations from the Brand Module, capped at **30% max** of the <code>minimumPool</code> size to avoid overshadowing the user's personal wardrobe.",
            },
            ensurepool: {
                t: "EnsureMinimumCandidatePool",
                m: "Belly Expansion Safeguard · pool_expansion.go",
                b: "Expansion algorithm activated if the hybrid retrieval output falls below <code>minimumPool</code>. Iterates through the user's wardrobe and appends extra clothes using 3 progressive relaxation tiers:<br>• <b>Tier 1 (Strict):</b> Matches hard filters (season, category) and must match fallback search terms (<code>MatchesFallbackLexicalTerms</code>).<br>• <b>Tier 2 (Relaxed):</b> Only needs to match hard filters (season and category slug).<br>• <b>Tier 3 (General):</b> Removes all matching conditions, retrieving any active clothing items.<br><i>Note:</i> All expansion tiers exclude items containing excluded terms (<code>MatchesFallbackExcludedTerms</code>).",
            },
            candidatepool: {
                t: "Candidate Pool",
                m: "rankingCandidates · Memory Slice",
                b: "Memory-stored candidate list (slice of <code>CandidateForRanking</code>). Serves as the sole context block passed to ranking and prompt-building stages. Preserves item origin metadata (<code>UserWardrobe</code> or <code>BrandItem</code>).",
            },
            rankcandidates: {
                t: "RankCandidates / ScoreCandidateItem",
                m: "Heuristic Scorer · ranking/ranking.go",
                b: "Scores each candidate item using a heuristic scoring formula starting with a base score of 1.0:<br>• <b>Style:</b> +0.3 if matching target style; -0.35 if matching avoided style.<br>• <b>Occasion:</b> +0.2 if matching occasion directly; +0.08 if matching via Taxonomy hierarchy.<br>• <b>Color Tone:</b> +0.2 for target color tone; -0.25 for avoided color tone; +0.1 for neutral colors (highly compatible).<br>• <b>Weather:</b> Jackets get +0.4 when cold, -0.3 when hot; rainwear gets +0.15 when raining. Applies penalty of -0.2 to -0.3 if matching avoided weather properties.<br>• <b>Lexical Boost:</b> Adds up to +0.5 based on raw keyword overlap matching.<br>• <b>Wear History:</b> Applies up to -0.4 penalty if worn recently (linear decay); rewards +0.15 for long-unworn or brand new clothes.<br>• <b>Retrieval Bonus:</b> Adds up to +0.25 based on hybrid search score and ranking.<br>• <b>Fallback Penalty:</b> Deducts points based on retrieval source relaxation (Strict: -0.1, Relaxed: -0.2, General: -0.35).",
            },
            diversify: {
                t: "DiversifyRankedCandidates",
                m: "Soft Cap Diversity Filter · ranking.go",
                b: "Guarantees recommendation variety sent to the LLM. Operates in 2 passes:<br>1. <b>Pass 1 (Soft Cap):</b> Scans the scored list and takes at most <b>3 items</b> per category (tops, bottoms, dresses...). Excess items are deferred to a <code>deferred</code> list.<br>2. <b>Pass 2 (Backfill):</b> If the candidate selection size is under the limit, pulls from the <code>deferred</code> list in ranking order until the pool is filled.",
            },
            buildprompt: {
                t: "BuildRecommendationPromptWithLimits",
                m: "Dynamic Prompt Assembly · synthesis/prompt.go",
                b: "Builds the final prompt sent to the LLM containing strict styling constraints (e.g. no dresses paired with tops/bottoms, outerwears on top) and user context.<br>• <b>Alias Encoding:</b> Compresses item IDs into short aliases (<code>A1</code>, <code>A2</code>, <code>A3</code>...), cutting token usage and completely eradicating LLM-generated hallucinated UUID errors.<br>• <b>Dynamic Budgeting:</b> If prompt length exceeds <code>PromptMaxCharacters</code>, the builder sequentially prunes candidates from the bottom of the ranked list until it fits within the token budget.",
            },
            generatetext: {
                t: "GenerateRecommendationText",
                m: "LLM Styling Synthesis · Shared AI Service",
                b: "Transmits prompt to the large language model (Gemini or OpenAI depending on configuration). Applies optimal parameters:<br>• <b>Temperature = 0.1:</b> Keeps model outputs stable, restricting stray creativity that could breach coordination styling constraints.<br>• <b>JSON Schema:</b> Forces <code>application/json</code> MIME type output coupled with a strict JSON schema definition to ensure the generated payload matches structural expectations.",
            },
            isllmsuccess: {
                t: "LLM Success Check",
                m: "Resilience Decision Diamond",
                b: "Critical routing node:<br>• <b>Success Branch:</b> If LLM returns valid structured JSON, routes directly to parser and mapper.<br>• <b>Failure Branch:</b> If LLM errors (rate limits, timeouts, connection issues, safety blocks), triggers <code>ClassifyFallbackTrace</code> for log telemetry and redirects immediately to the local fallback coordination engine to maintain 100% Availability.",
            },
            resolution: {
                t: "Response Validation & ID Resolution",
                m: "Sanitization & Alias Decoding · mapper/recommendation.go",
                b: "LLM output parser and sanitizer:<br>1. <b>JSON Sanitization:</b> Strips off markdown fencing and calls <code>ExtractFirstJSONObject</code> to extract the first valid JSON block.<br>2. <b>Alias Decoding:</b> Decodes aliases (<code>A1</code>, <code>A2</code>...) back to database UUIDs.<br>3. <b>Placeholder Filtering:</b> Rejects outputs containing AI mock values (e.g. <i>'string'</i>, <i>'uuid'</i>, <i>'role'</i>) via <code>ValidateOutfitRecommendationPayload</code>.<br>4. <b>DTO Mapping:</b> Maps real items, automatically removing redundant top/bottom items if the outfit selected a fullbody dress.",
            },
            hslmatching: {
                t: "RunLocalHSLMatching",
                m: "Local Heuristic HSL Color Harmony · fallback_hsl.go",
                b: "Rule-based fallback coordination engine bypassing AI to guarantee 100% Availability:<br>• <b>1. Category Partitioning:</b> Groups Candidates Pool items into separate sub-lists: <i>tops, bottoms, shoes, outerwears, dresses, and accessories</i> based on `CategorySlug` attributes.<br>• <b>2. Weather Parsing:</b> Analyzes the input weather string. If it contains cold/wet indicators (<code>'lạnh', 'mát', 'cold', 'cool', 'mưa', 'rain'</code>), sets `isCold = true` to enable outerwear inclusion.<br>• <b>3. Outfit Coordination:</b><br>&nbsp;&nbsp;- <i>Dress & Shoes:</i> If dresses and shoes exist, selects the first dress. Searches for the first shoe matching the dress's color. If `isCold = true`, also grabs the first color-matching outerwear.<br>&nbsp;&nbsp;- <i>Tops & Bottoms:</i> Iterates through top-bottom pairs to find the <b>first color harmony match</b>. Then, matches shoes and outerwears (if cold) to either the <b>primary top or bottom</b>.<br>&nbsp;&nbsp;- <i>Fallback Default:</i> If no pairs match, falls back to joining the first item of each category (Top, Bottom, Shoe).<br>• <b>4. HSL Color Matching rules (colorsMatch):</b> Two items match if they satisfy one of:<br>&nbsp;&nbsp;1. <i>Neutrals:</i> Lightness L &le; 10.0 (black), L &ge; 90.0 (white), or Saturation S &le; 10.0 (gray/muted). Neutrals are compatible with any color.<br>&nbsp;&nbsp;2. <i>Analogous:</i> Color angle distance &Delta;H = min(|H<sub>1</sub> - H<sub>2</sub>|, 360 - |H<sub>1</sub> - H<sub>2</sub>|) &lt; 30&deg; and lightness difference |L<sub>1</sub> - L<sub>2</sub>| &ge; 15.0 to prevent muddy color matching.<br>&nbsp;&nbsp;3. <i>Complementary:</i> Opposite sides of the color wheel: &Delta;H &isin; [165.0, 195.0].<br>• <b>5. Alternatives:</b> Each slot (e.g. Primary Top) includes up to <b>2 alternatives</b> from the same category to allow UI swapping.",
            },
            dtores: {
                t: "dto.RecommendedOutfitRes",
                m: "Final Output DTO",
                b: "Final client response containing the advice title, Vietnamese outfit description, and a structured array of outfit combinations (primary + alternatives) tagged with the <code>IsFallback</code> flag to trigger UI indicators.",
            },
        },
        FLOWS: {
            happy: {
                name: "Happy Path — Standard LLM RAG",
                steps: [
                    "Client sends a recommendation request containing preferences, weather, and target style.",
                    "<code>nlpParser.Parse</code> extracts semantic constraints from the input prompt.",
                    "<code>buildRetrievalQuery</code> rewrites queries for both Vector Similarity and Lexical searches.",
                    "<code>aiService.GenerateEmbeddings</code> vectorizes the search criteria.",
                    "<code>GetHybridCandidates</code> performs parallel search, combining vector and tag results via RRF.",
                    "Candidates are loaded into the <code>Candidate Pool</code>.",
                    "<code>RankCandidates</code> scores items based on style, weather, history, and source.",
                    "<code>DiversifyCandidates</code> filters outcomes to ensure a balanced category representation (Soft Cap).",
                    "<code>BuildRecomPrompt</code> constructs a structured prompt injecting candidate aliases (e.g., <code>A1</code>, <code>A2</code>).",
                    "<code>GenerateRecText</code> sends the prompt to the LLM (Gemini) to build outfits.",
                    "The LLM succeeds, returning outfit structures in valid JSON.",
                    "<code>Response Validation</code> parses the JSON and decodes alias IDs back to UUIDs.",
                    "Validated recommendations are returned to the client inside the DTO."
                ]
            },
            fallback: {
                name: "Resilient Fallback Path — Local HSL",
                steps: [
                    "Client sends a recommendation request containing preferences, weather, and target style.",
                    "<code>nlpParser.Parse</code> extracts semantic constraints.",
                    "<code>buildRetrievalQuery</code> rewrites the query parameters.",
                    "<code>aiService.GenerateEmbeddings</code> vectorizes the query.",
                    "<code>GetHybridCandidates</code> retrieves candidate items using Vector + FTS.",
                    "Candidates populate the <code>Candidate Pool</code>.",
                    "<code>RankCandidates</code> computes item heuristic scores.",
                    "<code>DiversifyCandidates</code> ensures category variety and keeps top scorers.",
                    "<code>BuildRecomPrompt</code> generates the LLM prompt.",
                    "LLM fails (rate limit, timeout, or safety block).",
                    "Decision branch registers the LLM failure and triggers fallback.",
                    "<code>RunLocalHSLMatching</code> immediately runs local HSL color harmony algorithms.",
                    "Fallback outfits are returned to the client with the <code>IsFallback = true</code> flag."
                ]
            },
            padding: {
                name: "Candidate Pool Expansion Flow",
                steps: [
                    "Triggers Vector and Lexical Search query processes.",
                    "Checks the total candidate count. If it is below the minimum limit.",
                    "Invokes the guard function <code>EnsureMinimumCandidatePool</code>.",
                    "Scans wardrobe at **Tier 1 (Strict)**: Hard filters + fallback lexical terms.",
                    "If still insufficient, scans at **Tier 2 (Relaxed)**: Hard filters only.",
                    "If still insufficient, scans at **Tier 3 (General)**: Pulls all active items.",
                    "Combines results and removes duplicates to complete the <code>Candidate Pool</code>."
                ]
            },
            hybrid: {
                name: "Hybrid Retrieval Flow",
                steps: [
                    "Receives the raw query keywords and query vector embeddings.",
                    "**Vector branch:** pgvector calculates Cosine distances using the HNSW index.",
                    "HNSW climbs down the graph layers to quickly route nearest neighbor items.",
                    "**Lexical branch:** GIN index scans matching lexemes in the <code>tsvector</code> column.",
                    "Scores matches using <code>ts_rank_cd</code> based on keyword cover density.",
                    "Normalizes and merges outputs using the RRF (Reciprocal Rank Fusion) algorithm.",
                    "Top RRF ranked candidates populate the Candidate Pool."
                ]
            }
        }
    }
};

const FLOWS = {
    happy: {
        edges: [
            "e-req-parse", "e-parse-rewrite", "e-rewrite-vector", "e-rewrite-hybrid",
            "e-vector-hybrid", "e-db-hybrid", "e-hybrid-pool", "e-pool-rank",
            "e-rank-diversify", "e-diversify-prompt", "e-prompt-synthesis",
            "e-synth-decide", "e-decide-valid", "e-valid-dto"
        ],
        nodes: [
            "request", "nlpparse", "buildquery", "generateembeddings", "getcandidates",
            "candidatepool", "rankcandidates", "diversify", "buildprompt",
            "generatetext", "isllmsuccess", "resolution", "dtores"
        ]
    },
    fallback: {
        edges: [
            "e-req-parse", "e-parse-rewrite", "e-rewrite-vector", "e-rewrite-hybrid",
            "e-vector-hybrid", "e-db-hybrid", "e-hybrid-pool", "e-pool-rank",
            "e-rank-diversify", "e-diversify-prompt", "e-prompt-synthesis",
            "e-synth-decide", "e-decide-hsl", "e-hsl-dto"
        ],
        nodes: [
            "request", "nlpparse", "buildquery", "generateembeddings", "getcandidates",
            "candidatepool", "rankcandidates", "diversify", "buildprompt",
            "generatetext", "isllmsuccess", "hslmatching", "dtores"
        ]
    },
    padding: {
        edges: [
            "e-rewrite-hybrid", "e-db-hybrid", "e-hybrid-ensure", "e-ensure-pool", "e-pool-rank"
        ],
        nodes: [
            "buildquery", "getcandidates", "ensurepool", "candidatepool", "rankcandidates"
        ]
    },
    hybrid: {
        edges: [
            "e-rewrite-vector", "e-rewrite-hybrid", "e-vector-hybrid", "e-db-hybrid", "e-hybrid-pool"
        ],
        nodes: [
            "buildquery", "generateembeddings", "getcandidates", "db_active", "candidatepool"
        ]
    }
};

const stage = document.getElementById("stage");
const nodes = document.querySelectorAll(".node");
const edges = document.querySelectorAll(".edge");
const elbls = document.querySelectorAll(".elbl");
const chips = document.querySelectorAll(".chip");
const fcap = document.getElementById("flowcap");
const detail = document.getElementById("detail");

function toggleCard(id) {
    const card = document.getElementById(id);
    const btn = card.querySelector(".minimize-btn");
    const isMinimized = card.classList.toggle("minimized");
    btn.textContent = isMinimized ? "+" : "−";
}

function expandCard(id) {
    const card = document.getElementById(id);
    const btn = card.querySelector(".minimize-btn");
    card.classList.remove("minimized");
    if (btn) btn.textContent = "−";
}

let currentLang = localStorage.getItem("lang") || "vi";

function updateLanguageUI() {
    const t = TRANSLATIONS[currentLang];
    
    // Update language toggle button label
    document.getElementById("langToggle").textContent = currentLang === "vi" ? "EN" : "VI";
    
    // Update navbar texts
    document.querySelector(".bar h1").innerHTML = t.barTitle;
    document.querySelector(".bar .sub").innerHTML = t.barSub;
    
    // Update chips labels
    const chipKeys = ["all", "happy", "fallback", "padding", "hybrid"];
    chips.forEach((c, idx) => {
        c.textContent = t.chips[chipKeys[idx]];
    });
    
    // Update sidebar default welcome / detail
    const selectedNode = document.querySelector(".node.sel");
    if (!selectedNode) {
        document.getElementById("d-title").textContent = t.welcomeTitle;
        document.getElementById("d-meta").textContent = t.overview;
        document.getElementById("d-body").innerHTML = t.welcomeBody;
    } else {
        const d = t.DETAIL[selectedNode.dataset.k];
        if (d) {
            document.getElementById("d-title").textContent = d.t;
            document.getElementById("d-meta").textContent = d.m;
            document.getElementById("d-body").innerHTML = d.b;
        }
    }
    
    // Update current active flow content if any
    const activeChip = document.querySelector(".chip.on");
    if (activeChip) {
        const flowKey = activeChip.dataset.flow;
        if (flowKey !== "all") {
            const f = t.FLOWS[flowKey];
            if (f) {
                document.getElementById("f-name").textContent = f.name;
                document.getElementById("f-steps").innerHTML = f.steps
                    .map((s) => `<li>${s}</li>`)
                    .join("");
            }
        } else {
            document.getElementById("f-name").textContent = t.flowcapName;
        }
    }
    
    // Update source label
    document.querySelector(".src").textContent = t.sourceLabel;
}

function setFlow(key) {
    chips.forEach((c) =>
        c.classList.toggle("on", c.dataset.flow === key)
    );
    edges.forEach((e) => e.classList.remove("lit"));
    elbls.forEach((l) => l.classList.remove("lit"));
    nodes.forEach((n) => n.classList.remove("lit"));

    if (key === "all") {
        stage.classList.remove("flowing");
        fcap.classList.remove("show");
        return;
    }

    const f = FLOWS[key];
    const t = TRANSLATIONS[currentLang];
    const tf = t.FLOWS[key];
    
    stage.classList.add("flowing");
    f.edges.forEach((id) =>
        document.getElementById(id)?.classList.add("lit")
    );
    elbls.forEach((l) => {
        if (f.edges.includes(l.dataset.e)) l.classList.add("lit");
    });
    f.nodes.forEach((k) => {
        document
            .querySelector(`.node[data-k="${k}"]`)
            ?.classList.add("lit");
    });

    document.getElementById("f-name").textContent = tf.name;
    document.getElementById("f-steps").innerHTML = tf.steps
        .map((s) => `<li>${s}</li>`)
        .join("");
    fcap.classList.add("show");
}

chips.forEach((c) =>
    c.addEventListener("click", () => setFlow(c.dataset.flow))
);

const svgEl = stage.querySelector("svg");

nodes.forEach((n) =>
    n.addEventListener("click", (e) => {
        e.stopPropagation();
        nodes.forEach((x) => x.classList.remove("sel"));
        n.classList.add("sel");
        svgEl.classList.add("has-selected");
        const t = TRANSLATIONS[currentLang];
        const d = t.DETAIL[n.dataset.k];
        if (!d) return;
        document.getElementById("d-title").textContent = d.t;
        document.getElementById("d-meta").textContent = d.m;
        document.getElementById("d-body").innerHTML = d.b;
        expandCard("detail");
    })
);

// Clear selection when clicking SVG background
svgEl.addEventListener("click", (e) => {
    if (!e.target.closest(".node")) {
        nodes.forEach((x) => x.classList.remove("sel"));
        svgEl.classList.remove("has-selected");
        const t = TRANSLATIONS[currentLang];
        document.getElementById("d-title").textContent = t.welcomeTitle;
        document.getElementById("d-meta").textContent = t.overview;
        document.getElementById("d-body").innerHTML = t.welcomeBody;
    }
});

document
    .getElementById("themeToggle")
    .addEventListener("click", () => {
        const dark =
            !document.documentElement.classList.contains("dark");
        document.documentElement.classList.toggle("dark", dark);
        localStorage.setItem("theme", dark ? "dark" : "light");
    });

document
    .getElementById("langToggle")
    .addEventListener("click", () => {
        currentLang = currentLang === "vi" ? "en" : "vi";
        localStorage.setItem("lang", currentLang);
        updateLanguageUI();
    });

// Initialize Language on load
updateLanguageUI();
