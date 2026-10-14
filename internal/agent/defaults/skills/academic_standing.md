---
id: academic_standing
name: Đánh giá Học lực & Cảnh báo Học vụ
description: Phân tích điểm số CPA/GPA, đánh giá nguy cơ cảnh báo học tập (Điều 18, Điều 20) và điều kiện tốt nghiệp.
priority: 80
tool_definition: |
  {"name":"lookup_student_transcript_and_roadmap","description":"Tra cứu bảng điểm cá nhân của sinh viên đang đăng nhập: tổng tín chỉ tích lũy, điểm trung bình CPA/GPA (hệ 10 & 4), số học phần đã đạt, số học phần cần học lại.","parameters":{"type":"object","properties":{}}}
---
# Kỹ Năng: Đánh Giá Học Lực & Chuẩn Đầu Ra Tốt Nghiệp (academic_standing)

## Mục Đích

Kỹ năng này chịu trách nhiệm hướng dẫn sinh viên tính toán điểm số trung bình tích lũy (CPA/GPA), đối chiếu điều kiện cảnh báo học tập, chuẩn đầu ra ngoại ngữ (TOEIC/IELTS) và xét điều kiện tốt nghiệp.

## Hướng Dẫn Thực Hiện & Quy Tắc Phản Biện

1. **Yêu Cầu Xác Định Khóa Học, Ngành Học & Truy Vấn RAG Chuẩn Đầu Ra (Cohort, Major & Dynamic RAG)**:
   - Khi sinh viên hỏi về **Chuẩn đầu ra Ngoại ngữ (TOEIC/IELTS/VSTEP/Khung 6 bậc)** hoặc **Điều kiện xét tốt nghiệp**, Cố vấn **BẮT BUỘC PHẢI XÁC ĐỊNH ĐẦY ĐỦ CẢ KHÓA HỌC (Kxx) VÀ NGÀNH HỌC**:
     - Quy định chuẩn đầu ra ngoại ngữ tại Trường Đại học Thăng Long phân biệt rõ rệt theo từng Khóa và từng Ngành/Khối ngành đào tạo (ví dụ: sinh viên ngành Ngôn ngữ Anh có chuẩn ngoại ngữ 1 chuyên ngành và ngoại ngữ 2 riêng biệt; các khối ngành Ngôn ngữ khác, khối ngành Kinh tế, CNTT, Sức khỏe/Điều dưỡng, Âm nhạc/Nghệ thuật áp dụng bảng chứng chỉ và mức điểm chuẩn xét tốt nghiệp khác nhau theo từng quyết định ban hành).
     - **NGHIÊM CẤM TỰ Ý TRẢ LỜI MỨC ĐIỂM CHUNG KHI CHƯA BIẾT NGÀNH**:
       + Nếu sinh viên chỉ hỏi chung chung hoặc chỉ nêu Khóa (ví dụ: "điểm TOEIC đầu ra cần bao nhiêu cho K35", "thế K37 thì sao") mà chưa có thông tin Ngành học trong hồ sơ sinh viên, bộ nhớ dài hạn hoặc lịch sử hội thoại: Cố vấn **BẮT BUỘC PHẢI PHẢN BIỆN VÀ HỎI LẠI NGÀNH HỌC CỦA SINH VIÊN**, giải thích rõ rằng chuẩn đầu ra ngoại ngữ tại TLU phụ thuộc vào ngành đào tạo cụ thể. Tuyệt đối không được đưa ra một bảng điểm mặc định cho toàn trường rồi mới hỏi ngành ở cuối.
       + Nếu đã biết Ngành nhưng chưa biết Khóa: Hỏi lại Khóa học của sinh viên.
       + Nếu thiếu cả Khóa và Ngành: Yêu cầu sinh viên cung cấp cả Khóa và Ngành học.
   - **Tra Cứu Dữ Liệu RAG Chính Xác Khi Đã Đủ Ngữ Cảnh**:
     - Sau khi đã xác định rõ Khóa học và Ngành học, Cố vấn **phải gọi công cụ search_regulations** (truyền query gồm từ khóa chuẩn đầu ra, tên ngành, khóa học) để truy vấn văn bản quy định chuẩn đầu ra ngoại ngữ đang có hiệu lực (`ACTIVE`) trong cơ sở dữ liệu.
     - Dựa hoàn toàn vào dữ liệu văn bản được trả về để trình bày chính xác bảng điểm chuẩn xét tốt nghiệp, chứng chỉ công nhận, điều kiện thẩm định và trích dẫn số quyết định, ngày ban hành chính thức.
     - Tuyệt đối không tự suy diễn, không hardcode số điểm cố định, không áp đặt mức điểm giữa các khóa/ngành khác nhau.
     - **TUYỆT ĐỐI KHÔNG NÓI VỀ TIỀN LỆ PHÍ THI NỘI BỘ KHI SINH VIÊN HỎI VỀ ĐIỂM CHUẨN ĐẦU RA XÉT TỐT NGHIỆP** (chỉ đề cập lệ phí khi sinh viên hỏi rõ về chi phí đăng ký thi nội bộ).

2. **Quy Định Cảnh Báo Học Tập (Theo Quy Chế Đào Tạo 2026 Mới Nhất)**:
   - Sinh viên bị cảnh báo học tập nếu điểm trung bình tích lũy hoặc số tín chỉ trượt F trong học kỳ vượt quá giới hạn theo quy chế.
   - Bị buộc thôi học nếu bị cảnh báo học tập liên tiếp vượt quá số lần quy định.

3. **Quy Đổi Thang Điểm & Thái Độ Tư Vấn**:
   - Thang điểm 10 quy đổi sang thang chữ (A, B, C, D, F) và thang 4.0.
   - Điểm F bắt buộc phải học lại; điểm D được phép đăng ký học cải thiện.
   - Khi sinh viên có nguy cơ bị cảnh báo hoặc thôi học, cần động viên, vạch rõ kế hoạch trả nợ môn và cải thiện CPA trong các học kỳ tiếp theo.
