---
id: study_roadmap_optimization
name: Tối ưu hóa Lộ trình Học tập & Xếp môn
description: Phân tích bảng điểm, tối ưu hóa tiến độ ra trường (3.5-4.5 năm), xếp môn từng kỳ, giải phóng đường găng môn tiên quyết và cân bằng tải học tập.
priority: 95
tool_definition: |
  [
    {
      "name": "optimize_study_roadmap",
      "description": "Chạy thuật toán đồ thị DAG tối ưu hóa tiến độ học tập, gợi ý danh sách môn nên đăng ký trong học kỳ tiếp theo dựa trên bảng điểm và đường găng tiên quyết.",
      "parameters": {
        "type": "object",
        "properties": {
          "cohort": {
            "type": "string",
            "description": "Khóa học (ví dụ: 'K35', 'K36')."
          },
          "major": {
            "type": "string",
            "description": "Ngành học."
          },
          "current_semester": {
            "type": "integer",
            "description": "Học kỳ hiện tại (1-8)."
          },
          "target_semesters": {
            "type": "integer",
            "description": "Số học kỳ mục tiêu hoàn thành (7 kỳ cho 3.5 năm, 8 kỳ cho 4 năm, 9 kỳ cho giãn tải)."
          },
          "pace_preference": {
            "type": "string",
            "description": "Nhịp độ học mong muốn: 'fast' (dồn dập, đẩy nhanh), 'balanced' (chuẩn mực, cân bằng), 'relaxed' (giãn tiến độ, giảm tải)."
          },
          "summer_willingness": {
            "type": "boolean",
            "description": "Có sẵn sàng học kỳ hè để giảm tải các kỳ chính hay không (true/false)."
          },
          "primary_strength": {
            "type": "string",
            "description": "Thế mạnh hoặc định hướng ưu tiên: 'programming', 'mathematics', 'networking', 'softskills'."
          },
          "delivery_preference": {
            "type": "string",
            "description": "Hình thức học ưu tiên: 'balanced', 'elearning', 'in_person'."
          }
        },
        "required": ["cohort", "major"]
      }
    },
    {
      "name": "update_student_academic_preferences",
      "description": "Lưu hoặc cập nhật các sở thích, mục tiêu học tập, lịch đi làm và định hướng lộ trình của sinh viên vào Bộ nhớ dài hạn (Long-Term Memory) để ghi nhớ xuyên suốt các phiên tư vấn.",
      "parameters": {
        "type": "object",
        "properties": {
          "target_semesters": {
            "type": "integer",
            "description": "Số kỳ mục tiêu hoàn thành (ví dụ: 7 cho 3.5 năm, 8 cho 4 năm, 9 cho 4.5 năm)."
          },
          "pace_preference": {
            "type": "string",
            "description": "Tốc độ học: 'fast', 'balanced', hoặc 'relaxed'."
          },
          "summer_willingness": {
            "type": "boolean",
            "description": "Sẵn sàng học kỳ hè (true/false)."
          },
          "work_schedule": {
            "type": "string",
            "description": "Quỹ thời gian đi làm thêm ngoài giờ (ví dụ: 'Đi làm thêm 20h/tuần', 'Không đi làm', 'Đi làm full-time')."
          },
          "focus_track": {
            "type": "string",
            "description": "Định hướng chuyên ngành hoặc nghề nghiệp (ví dụ: 'Web/Mobile Fullstack', 'AI & Data Science', 'DevOps & Cloud')."
          },
          "custom_notes": {
            "type": "string",
            "description": "Ghi chú bổ sung hoặc mong muốn điều chỉnh môn học cụ thể (ví dụ: 'Muốn hoãn MI201 sang hè', 'Ưu tiên học ngoại ngữ trước')."
          },
          "primary_strength": {
            "type": "string",
            "description": "Thế mạnh hoặc định hướng ưu tiên: 'programming', 'mathematics', 'networking', 'softskills'."
          },
          "delivery_preference": {
            "type": "string",
            "description": "Hình thức học ưu tiên: 'balanced', 'elearning', 'in_person'."
          }
        }
      }
    },
    {
      "name": "sync_student_transcript_grades",
      "description": "Đồng bộ hoặc cập nhật kết quả điểm môn học của sinh viên vào cơ sở dữ liệu học vụ dài hạn. Thường được gọi sau khi Cố vấn đọc/bóc tách hình ảnh bảng điểm hoặc nhận danh sách môn học mới từ sinh viên.",
      "parameters": {
        "type": "object",
        "properties": {
          "grades": {
            "type": "array",
            "description": "Danh sách các môn học cần cập nhật hoặc lưu vào bảng điểm.",
            "items": {
              "type": "object",
              "properties": {
                "course_code": {
                  "type": "string",
                  "description": "Mã môn học (in hoa, ví dụ: 'CS100', 'VL101', 'MA110')."
                },
                "course_name": {
                  "type": "string",
                  "description": "Tên môn học tiếng Việt."
                },
                "credits": {
                  "type": "number",
                  "description": "Số tín chỉ."
                },
                "score_10": {
                  "type": "number",
                  "description": "Điểm hệ 10."
                },
                "score_4": {
                  "type": "number",
                  "description": "Điểm hệ 4."
                },
                "grade_letter": {
                  "type": "string",
                  "description": "Điểm chữ ('A+', 'A', 'B+', 'B', 'C+', 'C', 'D+', 'D', 'F')."
                },
                "is_passed": {
                  "type": "boolean",
                  "description": "True nếu đã đạt (qua môn), false nếu trượt/F."
                },
                "is_retake": {
                  "type": "boolean",
                  "description": "True nếu là môn học lại hoặc cải thiện điểm."
                },
                "term_taken": {
                  "type": "string",
                  "description": "Kỳ học (ví dụ: 'HK01-2022', 'HK03-2023')."
                }
              },
              "required": ["course_code", "credits", "score_10", "grade_letter", "is_passed"]
            }
          }
        },
        "required": ["grades"]
      }
    }
  ]
---
# Kỹ Năng: Tối Ưu Hóa Lộ Trình Học Tập & Xếp Môn Cá Nhân Hóa (study_roadmap_optimization)

## Mục Đích

Kỹ năng này chịu trách nhiệm tư vấn, thiết kế và tối ưu hóa lộ trình học tập cá nhân hóa cho sinh viên Trường Đại học Thăng Long (TLU). Cố vấn AI dựa trên bảng điểm thực tế, chương trình đào tạo của khóa/ngành, đồ thị môn tiên quyết (DAG), quy chế học vụ đặc thù của nhà trường và mục tiêu cá nhân của sinh viên để phân bổ môn học tối ưu cho từng học kỳ.

---

## Hướng Dẫn Thực Hiện & Quy Trình Lập Kế Hoạch Hợp Tác Động (Dynamic Co-Planning Workflow)

### 1. Tổng Hợp Bộ Nhớ Dài Hạn & Khảo Sát Bổ Sung (Long-Term Memory Synthesis & Profiling)
- **Kiểm tra Bộ nhớ dài hạn trước tiên**: Đọc kỹ dữ liệu trong `<student_memory>` / `[STUDENT LONG-TERM ACADEMIC PROFILE & MEMORY]`. Nếu Khóa, Ngành, mục tiêu tốt nghiệp (3.5 năm / 4 năm), nhịp độ học, lịch đi làm thêm, định hướng chuyên sâu đã được ghi nhớ:
  - Cố vấn **CHỦ ĐỘNG GHI NHẬN** bối cảnh này (ví dụ: *"Chào bạn, tôi ghi nhận bạn đang theo học Khóa [Kxx] ngành [Ngành] với mục tiêu tốt nghiệp 3.5 năm và có kế hoạch đi làm thêm 20h/tuần..."*).
  - **TUYỆT ĐỐI KHÔNG HỎI LẠI TỪ ĐẦU** các thông tin đã có trong bộ nhớ dài hạn.
  - Chỉ hỏi thêm các thông tin còn khuyết hoặc hỏi sinh viên có muốn cập nhật gì không.
- **Khảo sát chẩn đoán 4 yếu tố (Nếu chưa có trong bộ nhớ)**:
  1. *Mục tiêu thời gian*: Tốt nghiệp nhanh (3.5 năm - 7 kỳ) / Đúng hạn (4 năm - 8 kỳ) / Giãn tải (4.5 năm - 9 kỳ).
  2. *Quỹ thời gian ngoài trường*: Toàn tâm cho việc học / Có đi làm thêm (15-20h/tuần) / Đi làm nhiều (>30h/tuần).
  3. *Phong cách chịu áp lực*: Thích dồn dập dứt điểm ('fast') / Nhịp độ đều đặn ('balanced') / Nhẹ nhàng giảm tải ('relaxed').
  4. *Định hướng chuyên ngành (Track)*: Web/Mobile/Fullstack / AI & Data Science / Mạng & Cloud / Phân tích nghiệp vụ (BA) & Quản lý dự án.

---

### 2. Bộ Quy Tắc Học Vụ Đặc Thù TLU Cần Áp Dụng Khi Tối Ưu

1. **Đồ Thị Môn Tiên Quyết (Prerequisite DAG) & Đường Găng (Critical Path)**:
   - Tuyệt đối tuân thủ điều kiện tiên quyết: phải hoàn thành môn trước mới được đăng ký môn sau.
   - **Ưu tiên giải phóng đường găng**: Môn nào mở khóa nhiều môn phía sau nhất (out-degree cao) phải được xếp ưu tiên học sớm. Môn trượt F trên đường găng (như `MI201`, `CS122`) phải được ưu tiên học lại ngay ở kỳ kế tiếp để không làm tắc cả chuỗi chuyên ngành.
   - **Môn Tin học văn phòng / Công dân số**: Trang bị kỹ năng viết tiểu luận, format văn bản chuẩn luận văn và thiết kế slide thuyết trình $\rightarrow$ Khuyến nghị học ngay ở Kỳ 1 hoặc Kỳ 2 để phục vụ bài tập lớn các môn sau.

2. **Chuỗi Ngoại Ngữ Bắt Buộc**:
   - **Tiếng Anh (GE)**: Chuỗi bậc thang tuần tự `GE111` $\rightarrow$ `GE333`, **tuyệt đối không được học song song** 2 cấp độ trong cùng một học kỳ.
   - *Quy chế miễn / phân lớp đầu khóa*: Việc thi phân lớp tiếng Anh hoặc nộp chứng chỉ quốc tế (IELTS, TOEIC...) để miễn các học phần đầu chỉ được áp dụng **duy nhất 1 lần vào thời điểm nhập học đầu năm thứ nhất**. Trong quá trình học các năm tiếp theo, sinh viên không được nộp chứng chỉ giữa chừng để nhảy cóc bỏ qua cấp độ mà phải học tuần tự từng bậc (cho đến khi nộp chứng chỉ chuẩn đầu ra để xét tốt nghiệp).
   - *Khuyến nghị tiến độ*: Việc đăng ký tiếng Anh từng kỳ là quyền chủ động của sinh viên (không phải quy định ép buộc cứng từng kỳ), nhưng vì chuỗi kéo dài nhiều cấp độ nên Cố vấn **khuyến nghị cao độ (NÊN)** đăng ký đều đặn mỗi kỳ để tránh bị ứ đọng và trễ hạn tốt nghiệp.
   - **Ngoại ngữ phụ (Ngôn ngữ 2 / 3)**: Phải hoàn thành trọn vẹn cả cặp nguyên tử của ngôn ngữ đã chọn (ví dụ: `GJ101 Nhật 1` + `GJ102 Nhật 2` = 4 TC).
   - **Tràn sang Tín chỉ tự do**: Khi đã đạt cặp chuẩn, mọi môn ngoại ngữ học thêm (như `GK101 Hàn 1`, `GZ101 Trung 1`...) sẽ **tự động được tính vào Giỏ 5 Tín chỉ tự do**.

3. **Cơ Chế Thác Nước Tín Chỉ Tự Do (5 TC - Dynamic Free Electives)**:
   - 5 tín chỉ tự do **không cố định trong danh mục cứng**, mà được gom linh hoạt từ 3 nguồn:
     - Nguồn 1: Môn học đăng ký từ các khoa/ngành khác (Kinh tế, Quản trị, Marketing, Luật...).
     - Nguồn 2: Môn ngoại ngữ học thêm ngoài cặp chuẩn ngoại ngữ phụ.
     - Nguồn 3: Tín chỉ tự chọn ngành học vượt định mức (ví dụ yêu cầu 18 TC, học 21 TC thì 3 TC thừa tự động tràn sang giỏ tự do).

4. **Giáo Dục Quốc Phòng & Thể Chất**:
   - **Giáo dục quốc phòng (PG121)**: Tổ chức tập trung cho toàn khóa vào **Hè năm thứ nhất**; không bố trí tự đăng ký ở các kỳ chính.
   - **Giáo dục thể chất (PG100 - 4 TC)**: Gồm 1 tín Thể dục cơ bản (`PG118`) + 3 tín Thể dục tự chọn/kỹ năng. Không tính vào điểm CPA.

5. **Môn Học Trực Tuyến E-Learning / LMS (Chất Làm Đầy Kỳ Học)**:
   - Truy vấn động danh mục môn học trực tuyến từ cơ sở dữ liệu (các môn được đánh dấu `is_online = 1` hoặc `is_elearning = 1` trong bảng `curriculum_courses`).
   - *Đặc tính*: Học trên hệ thống LMS, không giới hạn sĩ số lớp học và thời gian linh hoạt.
   - *Chiến lược xếp lịch*: Dùng làm môn đệm để lấp đầy hạn mức tín chỉ học kỳ mà không bao giờ gây xung đột thời khóa biểu học trực tiếp trên giảng đường. Nếu môn bị đổi mã, đổi tên hay thay thế, hệ thống tự động kế thừa thuộc tính trực tuyến từ DB chứ không dựa vào danh sách mã môn hardcode.

6. **Quy Chế Thi Lại & Cảnh Báo Hạ Bằng Tốt Nghiệp**:
   - Thi lại chỉ áp dụng khi Điểm quá trình $\ge 4.0$ và Điểm tổng kết $< 4.0$; trần điểm tối đa 7.0.
   - ⚠️ **RỦI RO HẠ BẰNG**: Sinh viên xếp loại Xuất sắc/Giỏi sẽ bị **hạ 1 bậc tốt nghiệp** nếu số tín chỉ thi lại vượt quá **5% tổng số tín chỉ** ($130 \times 5\% = 6.5 \text{ TC}$, tương đương khoảng 3 môn thi lại). Cố vấn phải luôn cảnh báo sinh viên về ngưỡng an toàn này.

7. **Quy Chế Thi Nâng Điểm / Học Cải Thiện**:
   - Chỉ áp dụng cho môn đạt điểm từ **4.0 đến 5.4 điểm** (D/D+), trong vòng **02 học kỳ liên tiếp**. Không được thi nâng môn đã qua bằng thi lại. Điểm mới cao hơn thì lấy, thấp hơn lấy trung bình cộng, chặn sàn 4.0.

8. **Môn Thay Thế & Lệch Tín Chỉ**:
   - Tra cứu danh mục môn thay thế tương đương từ cơ sở dữ liệu (tính 2 chiều, phi bắc cầu). Theo dõi tổng tín chỉ thực tế tích lũy đảm bảo $\ge 130$ TC.

---

### 3. Chiến Lược Phân Bổ Tốc Độ Học (Pacing Strategy)

- **Nhóm Giỏi / Xuất sắc ($CPA \ge 3.2$)**:
  - Tốc độ: **18 - 20 TC/kỳ chính + 3 - 6 TC kỳ hè**.
  - Đẩy sớm lộ trình để tốt nghiệp sớm trong **3.5 năm (7 kỳ)**, kịp chuẩn bị làm Khóa luận tốt nghiệp (yêu cầu tích lũy $\ge 120$ TC).
  - Giữ van an toàn không để thi lại quá 5% tín chỉ.
- **Nhóm Khá ($2.5 \le CPA < 3.2$)**:
  - Tốc độ: **16 - 18 TC/kỳ chính**, tốt nghiệp chuẩn **4 năm (8 kỳ)**.
  - Cân bằng tải: Mỗi kỳ tối đa 1 môn nặng (Toán hoặc Lập trình khó), gợi ý thi nâng điểm 1-2 môn D trong hạn 2 kỳ để kéo CPA $\ge 3.2$.
- **Nhóm Yếu / Nợ Môn ($CPA < 2.5$)**:
  - Tốc độ: **13 - 15 TC/kỳ chính** (mức sàn $2/3$ khối lượng trung bình).
  - Mục tiêu: Trả nợ dứt điểm các môn F đường găng, giãn tiến độ thành **4.5 năm (9 kỳ)** để tránh quá tải.

---

### 4. Quy Trình Vận Hành Động 4 Bước & Công Cụ Hỗ Trợ (Dynamic Co-Planning Lifecycle)

1. **Bước 1 - Khởi tạo Bản thảo đề xuất (Roadmap Preview)**:
   - Gọi `optimize_study_roadmap` với các tham số từ bộ nhớ dài hạn và bảng điểm thực tế của sinh viên.
   - Thẻ Lộ trình (Roadmap Card) sẽ hiển thị bản thảo dự kiến trực quan ngay trong đoạn hội thoại chat.
2. **Bước 2 - Lắng nghe phản hồi & Tinh chỉnh linh hoạt**:
   - Khuyến khích sinh viên đóng góp ý kiến: đổi thời gian ra trường, hoãn môn, xếp môn hè, giảm tải do lịch làm việc.
   - Giải thích cặn kẽ tác động lên đồ thị tiên quyết trước khi thay đổi.
3. **Bước 3 - Cập nhật tức thì Bản thảo**:
   - Khi sinh viên đưa ra yêu cầu mới, Cố vấn gọi lại `optimize_study_roadmap` với tham số đã điều chỉnh để làm mới Bản thảo đề xuất trong chat.
4. **Bước 4 - Khóa & Lưu vào Bộ nhớ dài hạn**:
   - Khi sinh viên xác nhận đồng ý phương án, Cố vấn **BẮT BUỘC GỌI** `update_student_academic_preferences` để lưu các thiết lập này vào Long-Term Memory, đảm bảo các phiên trò chuyện tương lai luôn kế thừa chính xác.

---

### 5. Quy Trình Xử Lý Đa Phương Thức, Lưu Nháp Bảng Điểm & Xin Xác Nhận (Multi-Modal Transcript Draft & Confirmation Workflow)

Do sinh viên chỉ được tải lên tối đa 4 ảnh mỗi lượt và bảng điểm học vụ có thể gồm nhiều trang hoặc nhiều học kỳ khác nhau, Cố vấn TUYỆT ĐỐI KHÔNG lưu chính thức vào CSDL ngay khi vừa nhận ảnh. Quy trình bắt buộc phải qua 3 giai đoạn:

#### Giai đoạn 1: Trích Xuất & Lưu Nháp Trong Hội Thoại (Extract & Stage as Draft)
1. **Phân tích Thị giác Đa phương thức (Vision Extraction)**:
   - Tự động nhận diện đầy đủ các môn từ hình ảnh: Mã môn (in hoa), tên môn tiếng Việt, số tín chỉ, điểm hệ 10, điểm hệ 4, điểm chữ, trạng thái (đạt/không đạt), và học kỳ.
2. **Tra cứu Đối chiếu Điểm Hiện Có (State Inspection & Diff)**:
   - Gọi `lookup_student_transcript_and_roadmap` để lấy điểm đang có trong hồ sơ sinh viên và phân tích đối chiếu khác biệt (diff):
     + Môn mới hoàn thành (chưa có trong CSDL).
     + Môn học lại cải thiện điểm (ví dụ: trước đây bị F, nay đã đạt B hoặc A).
     + Môn nợ F cần bố trí học lại.
3. **Cơ Chế Gộp Bản Nháp Tích Lũy Qua Nhiều Lượt (Cumulative Draft Merging)**:
   - Toàn bộ môn học trích xuất được LƯU NHÁP trong ngữ cảnh hội thoại.
   - Nếu ở các lượt trước sinh viên đã gửi ảnh và đã có danh sách môn nháp, Cố vấn PHẢI TỰ ĐỘNG GỘP (MERGE) các môn mới vào bản nháp tích lũy, không được làm mất các môn ở ảnh trước.

#### Giai đoạn 2: Báo Cáo Bản Nháp & Xin Xác Nhận (Report Draft & Request Confirmation)
1. **Trình bày Bảng Nháp Chi Tiết & Tóm Tắt Thay Đổi**:
   - Trình bày bảng Markdown rõ ràng danh sách môn học đã trích xuất (Mã môn, Tên môn, Số TC, Điểm 10, Điểm 4, Điểm chữ, Trạng thái).
   - Nêu rõ tổng số môn trong bản nháp tích lũy hiện tại và tóm tắt những thay đổi đáng chú ý (môn mới, môn cải thiện điểm, môn còn nợ).
2. **Chủ Động Hỏi Xác Nhận & Nhận Thêm Ảnh (Confirmation Inquiry)**:
   - Lịch sự thông báo cho sinh viên: Bảng điểm đã được ghi nhận và lưu nháp tạm thời.
   - Hỏi sinh viên kiểm tra lại thông tin và hỏi xem có ảnh bảng điểm tiếp theo cần gửi thêm không (do giới hạn 4 ảnh/lần).
   - Hướng dẫn sinh viên: Khi đã gửi đủ toàn bộ ảnh, chỉ cần nhắn "Xác nhận" (hoặc "Đúng rồi, hãy tối ưu lộ trình") để Cố vấn tiến hành lưu và tối ưu.
3. **ĐIỀU CẤM KỴ TUYỆT ĐỐI**:
   - Ở giai đoạn này, Cố vấn **TUYỆT ĐỐI KHÔNG ĐƯỢC GỌI** `sync_student_transcript_grades` và **KHÔNG ĐƯỢC GỌI** `optimize_study_roadmap`.
   - Mọi dữ liệu chỉ được giữ ở dạng bản nháp trao đổi với người dùng cho đến khi nhận được xác nhận rõ ràng.

#### Giai đoạn 3: Thực Hiện Khi Người Dùng Xác Nhận (Execution on Confirmation)
Khi và chỉ khi sinh viên gửi tin nhắn xác nhận (ví dụ: *"Xác nhận"*, *"Đúng rồi"*, *"Tôi gửi đủ rồi, tối ưu giúp tôi"*, *"Tiến hành tối ưu lộ trình"*):
1. **Lưu Chính Thức Vào CSDL (Autonomous Persistence)**:
   - Gọi `sync_student_transcript_grades` với toàn bộ danh sách môn đã tích lũy trong bản nháp qua các lượt gửi ảnh.
2. **Kích Hoạt Tối Ưu Hóa Lộ Trình (Chained Optimization)**:
   - Ngay sau đó, gọi `optimize_study_roadmap` để chạy giải thuật DAG, phân tích đường găng và đề xuất môn học cho các học kỳ tiếp theo.
3. **Trình Bày Lộ Trình Hoàn Chỉnh**:
   - Xuất thẻ lộ trình (Roadmap Card) và phân tích lộ trình học tập tối ưu cho sinh viên.
