package guardrail

import (
	"encoding/base64"
	"encoding/hex"
	"regexp"
	"strings"
)

// Decision defines the outcome of a guardrail safety evaluation.
type Decision string

const (
	DecisionAllow             Decision = "ALLOW"
	DecisionBlockedOutOfScope Decision = "BLOCKED_OUT_OF_SCOPE"
	DecisionBlockedInjection  Decision = "BLOCKED_INJECTION"
)

// CheckResult contains decision details and standard polite response if blocked.
type CheckResult struct {
	Decision Decision
	Reason   string
	Response string
}

const (
	defaultOutOfScopeReply = "Chào bạn, tôi là Trợ lý Cố vấn Học vụ & Đời sống Sinh viên Trường Đại học Thăng Long (TLUAgent). Tôi hỗ trợ giải đáp các vấn đề về quy chế đào tạo, chương trình học, tín chỉ, học phí, học bổng, chuẩn đầu ra, thủ tục hành chính, cũng như đời sống sinh viên, các câu lạc bộ (CLB), cơ sở vật chất và các thông tin chính thức đã được Nhà trường công bố công khai. Vấn đề bạn hỏi hiện nằm ngoài phạm vi thông tin và tư vấn của Nhà trường."
	defaultInjectionReply  = "Yêu cầu không hợp lệ. Trợ lý Cố vấn Học vụ & Đời sống Sinh viên Đại học Thăng Long hoạt động theo quy chuẩn bảo mật và chỉ hỗ trợ giải đáp quy chế học vụ, đời sống sinh viên và thông tin chính thức của Trường Đại học Thăng Long. Mọi cấu hình hệ thống và chỉ dẫn nội bộ được bảo vệ tuyệt đối."
)

var (
	base64BlockRegex = regexp.MustCompile(`[A-Za-z0-9+/]{20,}={0,2}`)
	hexBlockRegex    = regexp.MustCompile(`(?i)[0-9a-f]{30,}`)

	injectionRegexes = []*regexp.Regexp{
		// English
		regexp.MustCompile(`(?i)\b(?:ignore|disregard|forget|override|bypass)\s+.*?\b(?:instructions|rules|prompts|directives|constraints)\b`),
		regexp.MustCompile(`(?i)\b(?:repeat|reveal|show|print|output|display|dump|what\s+are|tell\s+me)\s+.*?\b(?:system\s+prompt|developer\s+instructions|initial\s+prompt|instructions\s+above|secret\s+rules|rule\.md|soul\.md)\b`),
		regexp.MustCompile(`(?i)\b(?:repeat\s+everything\s+above|verbatim\s+starting\s+from)\b`),
		regexp.MustCompile(`(?i)\b(?:dan\s+mode|jailbreak|developer\s+mode|uncensored\s+ai|evil\s+bot|oppo\s+mode|do\s+anything\s+now)\b`),
		regexp.MustCompile(`(?i)\b(?:you\s+are\s+(?:now\s+)?dan|broken\s+free\s+of\s+all\s+rules)\b`),

		// Vietnamese
		regexp.MustCompile(`(?i)(?:bỏ\s*qua|quên\s*đi|hủy\s*bỏ|ghi\s*đè)\s+.*?(?:chỉ\s*dẫn|quy\s*tắc|hướng\s*dẫn|lệnh|ràng\s*buộc)`),
		regexp.MustCompile(`(?i)(?:tiết\s*lộ|cho\s*xem|in\s*ra|hiển\s*thị|đọc\s*lại|xuất\s*ra)\s+.*?(?:system\s*prompt|prompt\s*gốc|chỉ\s*dẫn\s*khởi\s*tạo|lệnh\s*hệ\s*thống|hướng\s*dẫn\s*nội\s*bộ|cấu\s*hình\s*llm|quản\s*trị\s*viên)`),
		regexp.MustCompile(`(?i)(?:ai\s*bẻ\s*khóa|bẻ\s*khóa\s*tự\s*do|không\s*bị\s*ràng\s*buộc\s*bởi\s*(?:bất\s*kỳ\s*)?quy\s*(?:chế|tắc|định))`),

		// Chinese (Simplified & Traditional)
		regexp.MustCompile(`(?i)(?:忽略|无视|跳过|忘记).*?(?:所有|全部)?.*?(?:指令|指示|提示词|规则|约束)`),
		regexp.MustCompile(`(?i)(?:显示|打印|输出|展示|告诉我).*?(?:系统提示词|系统指令|原始指令|开发者指令|system\s*prompt)`),
		regexp.MustCompile(`(?i)(?:越狱|DAN模式|无限制模式|解除限制)`),

		// Russian
		regexp.MustCompile(`(?i)(?:игнорируй|забудь|обойти).*?(?:все)?.*?(?:инструкции|правила|промпт|ограничения)`),
		regexp.MustCompile(`(?i)(?:покажи|выведи|распечатай).*?(?:системный\s+промпт|исходные\s+инструкции|промпт\s+разработчика)`),
		regexp.MustCompile(`(?i)(?:режим\s+DAN|джейлбрейк|делай\s+все\s+что\s+угодно)`),

		// French
		regexp.MustCompile(`(?i)(?:ignore|oublie|contourne).*?(?:toutes\s+les)?.*?(?:instructions|règles|prompts|directives)`),
		regexp.MustCompile(`(?i)(?:affiche|montre|révèle).*?(?:prompt\s+système|instructions\s+initiales|instructions\s+du\s+développeur)`),

		// Spanish
		regexp.MustCompile(`(?i)(?:ignora|olvida|omite).*?(?:todas\s+las)?.*?(?:instrucciones|reglas|prompts|directivas)`),
		regexp.MustCompile(`(?i)(?:muestra|imprime|revela).*?(?:prompt\s+del\s+sistema|instrucciones\s+del\s+desarrollador)`),

		// Japanese
		regexp.MustCompile(`(?i)(?:これまでの指示を(?:すべて)?無視|システムプロンプトを表示|システムプロンプトを出力|脱獄モード)`),

		// German
		regexp.MustCompile(`(?i)(?:ignoriere|vergiss).*?(?:alle)?.*?(?:anweisungen|regeln|prompts|befehle)`),
		regexp.MustCompile(`(?i)(?:zeige|gib|drucke).*?(?:system\s*prompt|entwickleranweisungen)`),
		regexp.MustCompile(`(?i)(?:dan\s*modus|ohne\s*einschränkungen)`),

		// Korean
		regexp.MustCompile(`(?i)(?:이전의?\s*(?:모든\s*)?(?:지침|규칙|명령|프롬프트를?)\s*(?:무시|잊어|취소))`),
		regexp.MustCompile(`(?i)(?:시스템\s*프롬프트|개발자\s*지침|초기\s*명령).*(?:출력|표시|보여|알려)`),
		regexp.MustCompile(`(?i)(?:DAN\s*모드|탈옥\s*모드|제한\s*해제)`),

		// Arabic
		regexp.MustCompile(`(?i)(?:تجاهل|انسَ|تخطى).*?(?:جميع)?.*?(?:التعليمات|القواعد|الأوامر|المطالبات)`),
		regexp.MustCompile(`(?i)(?:اعرض|أظهر|اطبع).*?(?:موجه\s*النظام|تعليمات\s*المطور|system\s*prompt)`),
		regexp.MustCompile(`(?i)(?:وضع\s*DAN|كسر\s*الحماية|بلا\s*قيود)`),

		// Portuguese
		regexp.MustCompile(`(?i)(?:ignore|esqueça|desconsidere).*?(?:todas\s*as)?.*?(?:instruções|regras|prompts|diretrizes)`),
		regexp.MustCompile(`(?i)(?:mostre|exiba|revele|imprima).*?(?:prompt\s+do\s+sistema|instruções\s+iniciais)`),
		regexp.MustCompile(`(?i)(?:modo\s+DAN|desbloqueado|sem\s+regras)`),

		// Italian
		regexp.MustCompile(`(?i)(?:ignora|dimentica).*?(?:tutte\s+le)?.*?(?:istruzioni|regole|prompt|direttive)`),
		regexp.MustCompile(`(?i)(?:mostra|visualizza|rivela|stampa).*?(?:prompt\s+di\s+sistema|istruzioni\s+iniziali)`),
		regexp.MustCompile(`(?i)(?:modalità\s+DAN|senza\s+regole|jailbreak)`),

		// Hindi
		regexp.MustCompile(`(?i)(?:पिछले\s*(?:सभी\s*)?(?:निर्देशों|नियमों)\s*को\s*(?:अनदेखा|भूल))`),
		regexp.MustCompile(`(?i)(?:सिस्टम\s*प्रॉम्प्ट|डेवलपर\s*निर्देश).*(?:दिखाएं|प्रिंट|प्रदर्शित)`),
		regexp.MustCompile(`(?i)(?:DAN\s*मोड|बिना\s*प्रतिबंध)`),

		// Thai
		regexp.MustCompile(`(?i)(?:ละเว้น|ลืม|เพิกเฉย).*?(?:คำแนะนำ|กฎ|คำสั่ง|system\s*prompt)`),
		regexp.MustCompile(`(?i)(?:แสดง|พิมพ์|บอก).*?(?:system\s*prompt|คำแนะนำระบบ)`),
		regexp.MustCompile(`(?i)(?:โหมด\s*DAN|เจลเบรค|ปลดล็อก)`),

		// Indonesian / Malay
		regexp.MustCompile(`(?i)\b(?:abaikan|lupakan)\s+.*?\b(?:semua\s+)?(?:instruksi|aturan|perintah|arahan)\b`),
		regexp.MustCompile(`(?i)\b(?:tampilkan|perlihatkan|cetak)\s+.*?\b(?:system\s+prompt|instruksi\s+pengembang)\b`),
		regexp.MustCompile(`(?i)\b(?:mode\s+DAN|tanpa\s+batasan|jailbreak)\b`),

		// System tags / Markdown Delimiters
		regexp.MustCompile(`(?i)(?:<\/?(?:system|instruction|developer|system_prompt)>|\[SYSTEM(?:_INSTRUCTION)?\]|###\s*System\s*Instruction|\[SYSTEM\s+INSTRUCTION[^\]]*\])`),
	}

	romanceRegexes = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:tỏ\s*tình|tán\s*(?:gái|trai|crush|đổ|tỉnh)|người\s*yêu|thất\s*tình|chia\s*tay|hẹn\s*hò|tình\s*yêu|yêu\s*đương|tâm\s*sự\s*tình\s*cảm)`),
		regexp.MustCompile(`(?i)\b(?:dating|break\s*up|confess\s+love|love\s+advice|fall\s+in\s+love)\b`),
	}

	generalCodeRegexes = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(?:viết\s*(?:cho\s*tôi\s*)?code|viết\s*(?:cho\s*tôi\s*)?chương\s*trình|viết\s*script|code\s*hộ|làm\s*hộ\s*bài\s*code|hướng\s*dẫn\s*viết\s*(?:code|bot))\b`),
		regexp.MustCompile(`(?i)\b(?:viết\s*code|lập\s*trình|code\s*giúp)\s+.*?\b(?:python|c\+\+|java|javascript|c#|golang|bot\s*discord|game|web)\b`),
		regexp.MustCompile(`(?i)\b(?:code\s+game|game\s+rắn\s+săn\s+mồi|game\s+flappy|game\s+bắn\s+súng|game\s+cờ\s+caro)\b`),
		regexp.MustCompile(`(?i)\b(?:write\s+code\s+for|code\s+a\s+game|python\s+script\s+to|implement\s+a\s+binary\s+tree|write\s+a\s+python\s+program)\b`),
	}

	lifestyleRegexes = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(?:nấu\s*(?:món|ăn)|công\s*thức\s*nấu|món\s*(?:phở|bún|lẩu)|dự\s*đoán\s*kết\s*quả\s*(?:bóng\s*đá|xổ\s*số)|soi\s*kèo|cá\s*độ|cúp\s*c1|ngoại\s*hạng\s*anh)\b`),
		regexp.MustCompile(`(?i)\b(?:how\s+to\s+cook|recipe\s+for|football\s+prediction|betting\s+odds)\b`),
	}

	academicContextRegex = regexp.MustCompile(`(?i)(?:môn|học\s*phần|tín\s*chỉ|chuẩn\s*đầu\s*ra|ngành|chương\s*trình\s*đào\s*tạo|khoa|giảng\s*viên|đăng\s*ký\s*(?:môn|học\s*phần)|tiên\s*quyết|song\s*bằng|đồ\s*án|đề\s*tài|khóa\s*luận|thực\s*tập|tốt\s*nghiệp|bài\s*tập\s*lớn|kỳ\s*học|học\s*kỳ|hủy\s*(?:môn|học\s*phần)|hoãn\s*thi|học\s*lại|học\s*cải\s*thiện|cảnh\s*báo\s*học\s*vụ|bảng\s*điểm|điểm\s*(?:số|thi|gpa|rèn\s*luyện|tích\s*lũy|học\s*phần|f|d|c|b|a|\d)|học\s*phí|học\s*bổng|bảo\s*lưu|clb|câu\s*lạc\s*bộ|ký\s*túc\s*xá|ktx|phòng\s*đào\s*tạo|tiếp\s*sinh\s*viên|nhà\s*t|sinh\s*viên|thăng\s*long|tlu|nckh|nghiên\s*cứu\s*khoa\s*học|đời\s*sống|thư\s*viện|nhà\s*ăn|căn\s*tin|canteen|sân\s*bóng|nhà\s*thi\s*đấu|phòng\s*gym|y\s*tế|trạm\s*y\s*tế|bhyt|bảo\s*hiểm\s*y\s*tế|gửi\s*xe|vé\s*xe|wifi|hội\s*trường|vườn\s*sinh\s*viên|đoàn\s*thanh\s*niên|hội\s*sinh\s*viên|tình\s*nguyện|ngoại\s*khóa|phong\s*trào|sự\s*kiện|cuộc\s*thi|workshop|talkshow|lễ\s*khai\s*giảng|chào\s*tân|thông\s*báo|tin\s*tức|lịch\s*học|lịch\s*thi|thời\s*khóa\s*biểu|tkb|tuyển\s*sinh|liên\s*hệ|cổng\s*thông\s*tin|(?:^|[^\p{L}\p{N}])(?:thầy|cô)(?:[^\p{L}\p{N}]|$))`)
)

func checkEncodedPayloads(query string) bool {
	base64Matches := base64BlockRegex.FindAllString(query, -1)
	for _, match := range base64Matches {
		decodedBytes, err := base64.StdEncoding.DecodeString(match)
		if err != nil {
			continue
		}
		decodedStr := string(decodedBytes)
		for _, re := range injectionRegexes {
			if re.MatchString(decodedStr) {
				return true
			}
		}
	}

	hexMatches := hexBlockRegex.FindAllString(query, -1)
	for _, match := range hexMatches {
		decodedBytes, err := hex.DecodeString(match)
		if err != nil {
			continue
		}
		decodedStr := string(decodedBytes)
		for _, re := range injectionRegexes {
			if re.MatchString(decodedStr) {
				return true
			}
		}
	}

	return false
}

// CheckQuery inspects the user inquiry for safety violations, prompt injections, and out-of-scope topics.
func CheckQuery(query string) CheckResult {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return CheckResult{Decision: DecisionAllow}
	}

	if checkEncodedPayloads(trimmed) {
		return CheckResult{
			Decision: DecisionBlockedInjection,
			Reason:   "Obfuscated encoded (Base64/Hex) prompt injection pattern detected",
			Response: defaultInjectionReply,
		}
	}

	for _, re := range injectionRegexes {
		if re.MatchString(trimmed) {
			return CheckResult{
				Decision: DecisionBlockedInjection,
				Reason:   "Prompt injection, system prompt extraction, or jailbreak pattern detected",
				Response: defaultInjectionReply,
			}
		}
	}

	if academicContextRegex.MatchString(trimmed) {
		return CheckResult{Decision: DecisionAllow}
	}

	for _, re := range romanceRegexes {
		if re.MatchString(trimmed) {
			return CheckResult{
				Decision: DecisionBlockedOutOfScope,
				Reason:   "Romance, dating, or relationship topic is out of academic advisory scope",
				Response: defaultOutOfScopeReply,
			}
		}
	}

	for _, re := range lifestyleRegexes {
		if re.MatchString(trimmed) {
			return CheckResult{
				Decision: DecisionBlockedOutOfScope,
				Reason:   "Lifestyle, cooking, or sports prediction topic is out of academic advisory scope",
				Response: defaultOutOfScopeReply,
			}
		}
	}

	for _, re := range generalCodeRegexes {
		if re.MatchString(trimmed) {
			return CheckResult{
				Decision: DecisionBlockedOutOfScope,
				Reason:   "General programming or game development requests are outside academic advisory scope",
				Response: defaultOutOfScopeReply,
			}
		}
	}

	return CheckResult{Decision: DecisionAllow}
}

// SanitizeOutput verifies generated assistant reply to prevent unintended internal prompt leakage.
func SanitizeOutput(reply string) string {
	forbiddenPatterns := []string{
		"MANDATORY CITATION & VERIFICATION DIRECTIVE:",
		"STRICT ZERO-HALLUCINATION POLICY:",
		"# Căn Tính & Sứ Mệnh Cố Vấn Học Vụ",
		"# RETRIEVED REGULATORY DOCUMENTS",
		"[SYSTEM ADVISORY RULE]",
		"[SYSTEM INSTRUCTION",
		"<system_prompt>",
		"CONFIDENTIAL INTERNAL RULES:",
		"Lex Posterior Derogat Legi Priori",
	}

	for _, pat := range forbiddenPatterns {
		if strings.Contains(reply, pat) {
			return "Thông tin được trích xuất từ quy chế học vụ chính thức của Trường Đại học Thăng Long. Vui lòng tham khảo thông báo chính thức tại Phòng Đào tạo."
		}
	}

	return reply
}
