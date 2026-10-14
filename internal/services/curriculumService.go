package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/guardrail"
)

// CurriculumService manages structured academic programs, prerequisites, and substitutions.
type CurriculumService interface {
	GetCurriculum(ctx context.Context, cohort, majorName string) (*response.CurriculumResponse, []*response.CurriculumCourseResponse, error)
	SearchCourses(ctx context.Context, cohort, majorName, keyword string) ([]*response.CurriculumCourseResponse, error)
	GetSubstitutions(ctx context.Context, courseCode, cohort, major string) ([]*response.CourseSubstitutionResponse, []*response.CourseGroupSubstitutionResponse, error)
	GetPrerequisites(ctx context.Context, courseCode, cohort string) ([]*response.CoursePrerequisiteResponse, []*response.CoursePrerequisiteResponse, error)
	FormatCurriculumContext(courses []*response.CurriculumCourseResponse) string
	FormatSubstitutionsContext(courseCode string, subs []*response.CourseSubstitutionResponse, grps []*response.CourseGroupSubstitutionResponse) string
	FormatPrerequisitesContext(courseCode, cohort string, prereqs []*response.CoursePrerequisiteResponse, dependents []*response.CoursePrerequisiteResponse) string

	ListCohorts(ctx context.Context) ([]string, error)
	ListMajorsByCohort(ctx context.Context, cohort string) ([]*response.MajorResponse, error)
	ListAllMajors(ctx context.Context) ([]*response.MajorResponse, error)
	ListCoursesPaginated(ctx context.Context, cohort, major, knowledgeBlock, keyword string, semester, isElearning, page, limit int) ([]*response.CurriculumCourseResponse, int64, error)
	GetCourseByID(ctx context.Context, id string) (*response.CurriculumCourseResponse, error)
	CreateCourse(ctx context.Context, req *request.CreateCourseRequest) (*response.CurriculumCourseResponse, error)
	UpdateCourse(ctx context.Context, id string, req *request.UpdateCourseRequest) (*response.CurriculumCourseResponse, error)
	DeleteCourse(ctx context.Context, id string) error
	ListSubstitutionsPaginated(ctx context.Context, scopeCohort, scopeMajor, keyword string, page, limit int) ([]*response.CourseSubstitutionResponse, int64, error)
	GetSubstitutionByID(ctx context.Context, id string) (*response.CourseSubstitutionResponse, error)
	CreateSubstitution(ctx context.Context, req *request.CreateSubstitutionRequest) (*response.CourseSubstitutionResponse, error)
	UpdateSubstitution(ctx context.Context, id string, req *request.UpdateSubstitutionRequest) (*response.CourseSubstitutionResponse, error)
	DeleteSubstitution(ctx context.Context, id string) error
	GetStats(ctx context.Context) (*response.CurriculumStatsResponse, error)
	GetVisualizationData(ctx context.Context, cohort, majorName string) (*response.CurriculumVisualizationDataResponse, error)
	GetStudentPersonalRoadmap(ctx context.Context, userID, cohort, majorName string, grades []*response.StudentCourseGradeItem, optimizer RoadmapOptimizer) (*response.StudentPersonalRoadmapDataResponse, error)
	FindMajorInText(ctx context.Context, text string) string
}

type curriculumService struct {
	repo repositories.CurriculumRepository
}

// NewCurriculumService creates an instance of CurriculumService.
func NewCurriculumService(repo repositories.CurriculumRepository) CurriculumService {
	return &curriculumService{repo: repo}
}

// MajorResolution holds the dynamic database-resolved major mapping.
type MajorResolution struct {
	MajorName   string
	MajorCode   string
	FacultyCode string
	Aliases     string
}

func parseCohortNumber(cohort string) int64 {
	var num int64
	found := false
	for _, r := range cohort {
		if unicode.IsDigit(r) {
			num = num*10 + int64(r-'0')
			found = true
		} else if found {
			break
		}
	}
	return num
}

func matchMajorRow(m sqlc.ListAllMajorMetadataRow, rawInput, normalizedInput string) bool {
	if m.MajorCode != "" && strings.EqualFold(m.MajorCode, rawInput) {
		return true
	}
	if strings.EqualFold(m.MajorName, rawInput) {
		return true
	}
	if guardrail.FoldToASCII(m.MajorName) == normalizedInput {
		return true
	}
	if m.Aliases != "" {
		for _, alias := range strings.Split(m.Aliases, ",") {
			trimmedAlias := strings.TrimSpace(alias)
			if strings.EqualFold(trimmedAlias, rawInput) || guardrail.FoldToASCII(trimmedAlias) == normalizedInput {
				return true
			}
		}
	}
	return false
}

func buildMajorResolution(m sqlc.ListAllMajorMetadataRow, fallbackName string) MajorResolution {
	name := m.MajorName
	if name == "" {
		name = fallbackName
	}
	code := m.MajorCode
	if code == "" {
		code = name
	}
	faculty := m.FacultyCode
	if faculty == "" {
		faculty = code
	}
	return MajorResolution{
		MajorName:   name,
		MajorCode:   code,
		FacultyCode: faculty,
		Aliases:     m.Aliases,
	}
}

func resolveMajorMetadata(ctx context.Context, repo repositories.CurriculumRepository, cohort, majorInput string) MajorResolution {
	trimmed := strings.TrimSpace(majorInput)
	if trimmed == "" {
		return MajorResolution{}
	}

	metadataList, err := repo.ListAllMajorMetadata(ctx)
	if err != nil || len(metadataList) == 0 {
		mapping, mErr := repo.FindMajorMapping(ctx, cohort, trimmed)
		if mErr == nil && mapping != nil {
			row := sqlc.ListAllMajorMetadataRow{
				Cohort:      cohort,
				MajorName:   mapping.MajorName,
				MajorCode:   mapping.MajorCode,
				FacultyCode: mapping.FacultyCode,
				Aliases:     mapping.Aliases,
			}
			return buildMajorResolution(row, trimmed)
		}
		upper := strings.ToUpper(trimmed)
		return MajorResolution{
			MajorName:   trimmed,
			MajorCode:   upper,
			FacultyCode: upper,
		}
	}

	normalizedInput := guardrail.FoldToASCII(trimmed)
	normalizedInput = strings.TrimPrefix(normalizedInput, "nganh ")
	normalizedInput = strings.TrimPrefix(normalizedInput, "chuyen nganh ")
	normalizedInput = strings.TrimPrefix(normalizedInput, "khoa ")
	normalizedInput = strings.TrimSpace(normalizedInput)

	cohortTrimmed := strings.ToUpper(strings.TrimSpace(cohort))

	if cohortTrimmed != "" {
		for _, m := range metadataList {
			if strings.EqualFold(m.Cohort, cohortTrimmed) {
				if matchMajorRow(m, trimmed, normalizedInput) {
					return buildMajorResolution(m, trimmed)
				}
			}
		}
	}

	for _, m := range metadataList {
		if matchMajorRow(m, trimmed, normalizedInput) {
			return buildMajorResolution(m, trimmed)
		}
	}

	for _, m := range metadataList {
		normName := guardrail.FoldToASCII(m.MajorName)
		if strings.Contains(normName, normalizedInput) || strings.Contains(normalizedInput, normName) {
			return buildMajorResolution(m, trimmed)
		}
	}

	mapping, mErr := repo.FindMajorMapping(ctx, cohort, trimmed)
	if mErr == nil && mapping != nil {
		row := sqlc.ListAllMajorMetadataRow{
			Cohort:      cohort,
			MajorName:   mapping.MajorName,
			MajorCode:   mapping.MajorCode,
			FacultyCode: mapping.FacultyCode,
			Aliases:     mapping.Aliases,
		}
		return buildMajorResolution(row, trimmed)
	}

	upper := strings.ToUpper(trimmed)
	return MajorResolution{
		MajorName:   trimmed,
		MajorCode:   upper,
		FacultyCode: upper,
		Aliases:     "",
	}
}

func (s *curriculumService) GetCurriculum(ctx context.Context, cohort, majorName string) (*response.CurriculumResponse, []*response.CurriculumCourseResponse, error) {
	c := strings.ToUpper(strings.TrimSpace(cohort))
	res := resolveMajorMetadata(ctx, s.repo, c, majorName)
	curr, err := s.repo.GetCurriculumByCohortAndMajor(ctx, c, res.MajorName)
	if err != nil && res.MajorName != majorName {
		curr, err = s.repo.GetCurriculumByCohortAndMajor(ctx, c, majorName)
	}
	if err != nil {
		return nil, nil, err
	}
	courses, err := s.repo.GetCurriculumCoursesByCohortAndMajor(ctx, c, curr.MajorName)
	if err != nil {
		return curr.ToResponse(), nil, err
	}
	return curr.ToResponse(), models.CurriculumCoursesToResponse(courses), nil
}

func (s *curriculumService) SearchCourses(ctx context.Context, cohort, majorName, keyword string) ([]*response.CurriculumCourseResponse, error) {
	c := strings.ToUpper(strings.TrimSpace(cohort))
	res := resolveMajorMetadata(ctx, s.repo, c, majorName)
	courses, err := s.repo.SearchCurriculumCourses(ctx, c, res.MajorName, keyword)
	if (err != nil || len(courses) == 0) && res.MajorName != majorName {
		courses, err = s.repo.SearchCurriculumCourses(ctx, c, majorName, keyword)
	}
	return models.CurriculumCoursesToResponse(courses), err
}

func (s *curriculumService) GetSubstitutions(ctx context.Context, courseCode, cohort, major string) ([]*response.CourseSubstitutionResponse, []*response.CourseGroupSubstitutionResponse, error) {
	code := strings.ToUpper(strings.TrimSpace(courseCode))
	res := resolveMajorMetadata(ctx, s.repo, cohort, major)
	cohortNum := parseCohortNumber(cohort)

	directSubs, err := s.repo.FindDirectSubstitutions(ctx, code, cohortNum, res.MajorCode, res.FacultyCode)
	if err != nil {
		return nil, nil, err
	}

	var groupSubs []*models.CourseGroupSubstitution
	if grps, err := s.repo.ListGroupSubstitutions(ctx, cohortNum, res.MajorCode, res.FacultyCode); err == nil {
		for _, g := range grps {
			prefix := ""
			if len(code) >= 2 {
				prefix = code[:2]
			}
			if prefix != "" && strings.Contains(g.AllowedPrefixes, prefix) {
				groupSubs = append(groupSubs, g)
			}
		}
	}

	return models.CourseSubstitutionsToResponse(directSubs), models.CourseGroupSubstitutionsToResponse(groupSubs), nil
}

func (s *curriculumService) GetPrerequisites(ctx context.Context, courseCode, cohort string) ([]*response.CoursePrerequisiteResponse, []*response.CoursePrerequisiteResponse, error) {
	code := strings.ToUpper(strings.TrimSpace(courseCode))
	c := strings.ToUpper(strings.TrimSpace(cohort))

	prereqs, err := s.repo.ListPrerequisitesForCourse(ctx, c, code)
	if err != nil {
		return nil, nil, err
	}

	dependents, err := s.repo.ListDependentCourses(ctx, c, code)
	if err != nil {
		return models.CoursePrerequisitesToResponse(prereqs), nil, err
	}

	return models.CoursePrerequisitesToResponse(prereqs), models.CoursePrerequisitesToResponse(dependents), nil
}

func (s *curriculumService) FormatCurriculumContext(courses []*response.CurriculumCourseResponse) string {
	if len(courses) == 0 {
		return "Không tìm thấy danh sách học phần trong cơ sở dữ liệu."
	}

	var sb strings.Builder
	sb.WriteString("| Mã HP | Tên học phần | Số TC | Hệ số | Tiên quyết | Khối kiến thức |\n")
	sb.WriteString("| :--- | :--- | :--- | :--- | :--- | :--- |\n")
	for _, c := range courses {
		fmt.Fprintf(&sb, "| %s | %s | %.1f | %.1f | %s | %s |\n",
			c.CourseCode, c.CourseName, c.Credits, c.Weight, c.PrerequisitesText, c.KnowledgeBlock)
	}
	return sb.String()
}

func (s *curriculumService) FormatSubstitutionsContext(courseCode string, subs []*response.CourseSubstitutionResponse, grps []*response.CourseGroupSubstitutionResponse) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# KẾT QUẢ TRA CỨU MÔN THAY THẾ CHO HỌC PHẦN: %s\n\n", courseCode)

	if len(subs) == 0 && len(grps) == 0 {
		sb.WriteString("Không có môn thay thế hoặc tương đương nào được quy định cho học phần này trong phạm vi khóa/ngành được chọn.\n")
		return sb.String()
	}

	if len(subs) > 0 {
		sb.WriteString("## Danh Mục Học Phần Thay Thế Trực Tiếp (Hai Chiều, Phi Bắc Cầu):\n")
		for idx, sub := range subs {
			fmt.Fprintf(&sb, "%d. **%s** (%s, %.1f TC) có thể thay thế bằng **%s** (%s, %.1f TC) [Khóa: %s, Ngành: %s, Căn cứ: %s]\n",
				idx+1, sub.OriginalCourseCode, sub.OriginalCourseName, sub.CreditsOriginal,
				sub.ReplacementCourseCode, sub.ReplacementCourseName, sub.CreditsReplacement,
				sub.ScopeCohort, sub.ScopeMajor, sub.SourceReference)
		}
		sb.WriteString("\n*Lưu ý nguyên tắc: Môn thay thế có tính 2 chiều giữa hai môn trên, nhưng TUYỆT ĐỐI KHÔNG có tính bắc cầu sang các môn học khác.*\n\n")
	}

	if len(grps) > 0 {
		sb.WriteString("## Quy Tắc Thay Thế Tín Chỉ Nhóm Chuyên Ngành:\n")
		for idx, g := range grps {
			fmt.Fprintf(&sb, "%d. %s -> %s (Khóa: %s, Ngành: %s)\n",
				idx+1, g.TargetDescription, g.ReplacementRuleDescription, g.ScopeCohort, g.ScopeMajor)
		}
	}

	return sb.String()
}

func (s *curriculumService) FormatPrerequisitesContext(courseCode, cohort string, prereqs []*response.CoursePrerequisiteResponse, dependents []*response.CoursePrerequisiteResponse) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# KẾT QUẢ TRA CỨU ĐIỀU KIỆN TIÊN QUYẾT: %s (%s)\n\n", courseCode, cohort)

	if len(prereqs) == 0 {
		sb.WriteString("- Môn học này **KHÔNG yêu cầu môn tiên quyết** trước khi đăng ký.\n\n")
	} else {
		sb.WriteString("## Môn Học Tiên Quyết Bắt Buộc Hoàn Thành Trước:\n")
		for idx, p := range prereqs {
			gradeNote := ""
			if p.MinGrade > 0 {
				gradeNote = fmt.Sprintf(" (yêu cầu điểm >= %.1f)", p.MinGrade)
			}
			fmt.Fprintf(&sb, "%d. **%s**%s\n", idx+1, p.PrerequisiteCourseCode, gradeNote)
		}
		sb.WriteString("\n")
	}

	if len(dependents) > 0 {
		sb.WriteString("## Các Môn Học Nối Tiếp (Yêu cầu môn này làm tiên quyết):\n")
		for idx, d := range dependents {
			fmt.Fprintf(&sb, "%d. **%s**\n", idx+1, d.CourseCode)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (s *curriculumService) ListCohorts(ctx context.Context) ([]string, error) {
	return s.repo.ListCohorts(ctx)
}

func (s *curriculumService) ListMajorsByCohort(ctx context.Context, cohort string) ([]*response.MajorResponse, error) {
	c := strings.ToUpper(strings.TrimSpace(cohort))
	rows, err := s.repo.ListMajorsByCohort(ctx, c)
	if err != nil {
		return nil, err
	}
	out := make([]*response.MajorResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, &response.MajorResponse{MajorName: r.MajorName, MajorCode: r.MajorCode})
	}
	return out, nil
}

func (s *curriculumService) ListAllMajors(ctx context.Context) ([]*response.MajorResponse, error) {
	rows, err := s.repo.ListAllMajors(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*response.MajorResponse, 0, len(rows))
	seenCodes := make(map[string]int)
	for _, r := range rows {
		code := strings.TrimSpace(r.MajorCode)
		if code != "" {
			if idx, exists := seenCodes[code]; exists {
				if isCanonicalMajorName(code, r.MajorName) {
					out[idx].MajorName = r.MajorName
				}
				continue
			}
			seenCodes[code] = len(out)
		}
		out = append(out, &response.MajorResponse{MajorName: r.MajorName, MajorCode: code})
	}
	return out, nil
}

func isCanonicalMajorName(code, name string) bool {
	norm := guardrail.FoldToASCII(name)
	switch strings.ToUpper(code) {
	case "TE":
		return norm == "he thong thong tin"
	case "TC":
		return norm == "mang may tinh va truyen thong du lieu"
	case "DL":
		return norm == "quan tri dich vu du lich va lu hanh"
	default:
		return false
	}
}

func (s *curriculumService) ListCoursesPaginated(ctx context.Context, cohort, major, knowledgeBlock, keyword string, semester, isElearning, page, limit int) ([]*response.CurriculumCourseResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	c := strings.ToUpper(strings.TrimSpace(cohort))
	m := ""
	if major != "" {
		res := resolveMajorMetadata(ctx, s.repo, c, major)
		m = res.MajorName
	}

	params := sqlc.ListCoursesPaginatedParams{
		Cohort:            c,
		MajorName:         m,
		KnowledgeBlock:    knowledgeBlock,
		SuggestedSemester: int64(semester),
		IsElearning:       int64(isElearning),
		Keyword:           keyword,
		LimitVal:          int64(limit),
		OffsetVal:         int64(offset),
	}
	courses, err := s.repo.ListCoursesPaginated(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	countParams := sqlc.CountCoursesFilteredParams{
		Cohort:            c,
		MajorName:         m,
		KnowledgeBlock:    knowledgeBlock,
		SuggestedSemester: int64(semester),
		IsElearning:       int64(isElearning),
		Keyword:           keyword,
	}
	total, err := s.repo.CountCoursesFiltered(ctx, countParams)
	if err != nil {
		return nil, 0, err
	}
	return models.CurriculumCoursesToResponse(courses), total, nil
}

func (s *curriculumService) GetCourseByID(ctx context.Context, id string) (*response.CurriculumCourseResponse, error) {
	course, err := s.repo.GetCurriculumCourseByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return course.ToResponse(), nil
}

// cleanPrerequisitesText normalizes and removes trailing conjunctions or punctuation from prerequisite strings.
func cleanPrerequisitesText(text string) string {
	s := strings.TrimSpace(text)
	s = strings.TrimRight(s, " ,;")
	for strings.HasSuffix(strings.ToLower(s), " ho\u1eb7c") || strings.HasSuffix(strings.ToLower(s), " or") {
		s = strings.TrimSpace(s[:len(s)-len(" ho\u1eb7c")])
		s = strings.TrimRight(s, " ,;")
	}
	return s
}

func (s *curriculumService) CreateCourse(ctx context.Context, req *request.CreateCourseRequest) (*response.CurriculumCourseResponse, error) {
	courseCode := strings.ToUpper(strings.TrimSpace(req.CourseCode))
	courseName := strings.TrimSpace(req.CourseName)
	cohort := strings.TrimSpace(req.Cohort)
	major := strings.TrimSpace(req.MajorName)
	cleanCohort := strings.ToLower(strings.ReplaceAll(cohort, " ", "_"))
	cleanMajor := strings.ToLower(strings.ReplaceAll(major, " ", "_"))
	cleanCourseCode := strings.ToLower(strings.ReplaceAll(courseCode, " ", "_"))
	id := fmt.Sprintf("cc_%s_%s_%s_%d", cleanCohort, cleanMajor, cleanCourseCode, req.SuggestedSemester)
	curriculumID := fmt.Sprintf("curr_%s_%s", cleanCohort, cleanMajor)

	isEl := int64(0)
	if req.IsElearning {
		isEl = 1
	}

	params := sqlc.UpsertCurriculumCourseParams{
		ID:                id,
		CurriculumID:      curriculumID,
		Cohort:            cohort,
		MajorName:         major,
		CourseCode:        courseCode,
		CourseName:        courseName,
		Credits:           req.Credits,
		Weight:            req.Weight,
		Hours:             req.Hours,
		KnowledgeBlock:    req.KnowledgeBlock,
		PrerequisitesText: cleanPrerequisitesText(req.PrerequisitesText),
		SuggestedSemester: int64(req.SuggestedSemester),
		IsElearning:       isEl,
	}

	curr, err := s.repo.GetCurriculumByCohortAndMajor(ctx, params.Cohort, params.MajorName)
	if err == nil && curr != nil {
		params.CurriculumID = curr.ID
	} else {
		createdCurr, err := s.repo.UpsertCurriculum(ctx, sqlc.UpsertCurriculumParams{
			ID:           curriculumID,
			Cohort:       params.Cohort,
			AcademicYear: "",
			MajorName:    params.MajorName,
		})
		if err == nil && createdCurr != nil {
			params.CurriculumID = createdCurr.ID
		}
	}

	course, err := s.repo.UpsertCurriculumCourse(ctx, params)
	if err != nil {
		return nil, err
	}
	return course.ToResponse(), nil
}

func (s *curriculumService) UpdateCourse(ctx context.Context, id string, req *request.UpdateCourseRequest) (*response.CurriculumCourseResponse, error) {
	isEl := int64(0)
	if req.IsElearning {
		isEl = 1
	}
	params := sqlc.UpdateCurriculumCourseParams{
		ID:                id,
		CourseName:        strings.TrimSpace(req.CourseName),
		Credits:           req.Credits,
		Weight:            req.Weight,
		Hours:             req.Hours,
		KnowledgeBlock:    req.KnowledgeBlock,
		PrerequisitesText: cleanPrerequisitesText(req.PrerequisitesText),
		SuggestedSemester: int64(req.SuggestedSemester),
		IsElearning:       isEl,
	}
	course, err := s.repo.UpdateCurriculumCourse(ctx, params)
	if err != nil {
		return nil, err
	}
	return course.ToResponse(), nil
}

func (s *curriculumService) DeleteCourse(ctx context.Context, id string) error {
	return s.repo.DeleteCurriculumCourse(ctx, id)
}

func (s *curriculumService) ListSubstitutionsPaginated(ctx context.Context, scopeCohort, scopeMajor, keyword string, page, limit int) ([]*response.CourseSubstitutionResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	cohortNum := parseCohortNumber(scopeCohort)
	if strings.EqualFold(strings.TrimSpace(scopeCohort), "ALL") {
		cohortNum = 0
	}

	rawMajor := strings.TrimSpace(scopeMajor)
	effectiveMajor := rawMajor
	if strings.EqualFold(effectiveMajor, "ALL") {
		effectiveMajor = ""
		rawMajor = ""
	}

	resolved := resolveMajorMetadata(ctx, s.repo, scopeCohort, rawMajor)
	if effectiveMajor != "" && resolved.MajorCode != "" {
		effectiveMajor = resolved.MajorCode
	}

	aliases := resolved.Aliases
	if effectiveMajor != "" && strings.EqualFold(rawMajor, resolved.FacultyCode) && !strings.EqualFold(rawMajor, resolved.MajorCode) {
		metadataList, err := s.repo.ListAllMajorMetadata(ctx)
		if err == nil {
			var extraAliases []string
			for _, m := range metadataList {
				if strings.EqualFold(m.FacultyCode, rawMajor) {
					if m.MajorCode != "" {
						extraAliases = append(extraAliases, m.MajorCode)
					}
					if m.Aliases != "" {
						extraAliases = append(extraAliases, m.Aliases)
					}
				}
			}
			if len(extraAliases) > 0 {
				if aliases != "" {
					aliases = aliases + "," + strings.Join(extraAliases, ",")
				} else {
					aliases = strings.Join(extraAliases, ",")
				}
			}
		}
	}

	params := sqlc.ListSubstitutionsPaginatedParams{
		CohortNum:    cohortNum,
		ScopeMajor:   effectiveMajor,
		FacultyCode:  resolved.FacultyCode,
		MajorAliases: aliases,
		Keyword:      strings.TrimSpace(keyword),
		LimitVal:     int64(limit),
		OffsetVal:    int64(offset),
	}
	subs, err := s.repo.ListSubstitutionsPaginated(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	countParams := sqlc.CountSubstitutionsFilteredParams{
		CohortNum:    cohortNum,
		ScopeMajor:   effectiveMajor,
		FacultyCode:  resolved.FacultyCode,
		MajorAliases: aliases,
		Keyword:      strings.TrimSpace(keyword),
	}
	total, err := s.repo.CountSubstitutionsFiltered(ctx, countParams)
	if err != nil {
		return nil, 0, err
	}
	return models.CourseSubstitutionsToResponse(subs), total, nil
}

func (s *curriculumService) GetSubstitutionByID(ctx context.Context, id string) (*response.CourseSubstitutionResponse, error) {
	sub, err := s.repo.GetSubstitutionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return sub.ToResponse(), nil
}

func (s *curriculumService) CreateSubstitution(ctx context.Context, req *request.CreateSubstitutionRequest) (*response.CourseSubstitutionResponse, error) {
	origCode := strings.ToUpper(strings.TrimSpace(req.OriginalCourseCode))
	repCode := strings.ToUpper(strings.TrimSpace(req.ReplacementCourseCode))
	scopeCohort := strings.TrimSpace(req.ScopeCohort)
	minC := int64(req.MinCohort)
	maxC := int64(req.MaxCohort)
	if minC == 0 && maxC == 0 {
		minC, maxC = repositories.ParseCohortRange(scopeCohort)
	} else if maxC == 0 {
		maxC = 999
	}
	if scopeCohort == "" {
		if minC <= 1 && maxC >= 999 {
			scopeCohort = "ALL"
		} else if minC > 0 && maxC >= 999 {
			scopeCohort = fmt.Sprintf("K%d-K39", minC)
		} else if minC > 0 && maxC > 0 {
			if minC == maxC {
				scopeCohort = fmt.Sprintf("K%d", minC)
			} else {
				scopeCohort = fmt.Sprintf("K%d-K%d", minC, maxC)
			}
		}
	}
	cleanOrig := strings.ToLower(strings.ReplaceAll(origCode, " ", "_"))
	cleanRep := strings.ToLower(strings.ReplaceAll(repCode, " ", "_"))
	cleanCohort := strings.ToLower(strings.ReplaceAll(scopeCohort, " ", "_"))
	if cleanCohort == "" {
		cleanCohort = fmt.Sprintf("k%d_k%d", minC, maxC)
	}
	id := fmt.Sprintf("sub_%s_%s_%s", cleanOrig, cleanRep, cleanCohort)

	isBi := int64(0)
	if req.IsBidirectional {
		isBi = 1
	}
	isTr := int64(0)
	if req.IsTransitive {
		isTr = 1
	}

	params := sqlc.UpsertCourseSubstitutionParams{
		ID:                    id,
		OriginalCourseCode:    origCode,
		OriginalCourseName:    strings.TrimSpace(req.OriginalCourseName),
		ReplacementCourseCode: repCode,
		ReplacementCourseName: strings.TrimSpace(req.ReplacementCourseName),
		ScopeCohort:           scopeCohort,
		ScopeMajor:            req.ScopeMajor,
		MinCohort:             minC,
		MaxCohort:             maxC,
		IsBidirectional:       isBi,
		IsTransitive:          isTr,
		CreditsOriginal:       req.CreditsOriginal,
		CreditsReplacement:    req.CreditsReplacement,
		RuleType:              req.RuleType,
		RuleNote:              req.RuleNote,
		SourceReference:       req.SourceReference,
	}
	sub, err := s.repo.UpsertCourseSubstitution(ctx, params)
	if err != nil {
		return nil, err
	}
	return sub.ToResponse(), nil
}

func (s *curriculumService) UpdateSubstitution(ctx context.Context, id string, req *request.UpdateSubstitutionRequest) (*response.CourseSubstitutionResponse, error) {
	origCode := strings.ToUpper(strings.TrimSpace(req.OriginalCourseCode))
	repCode := strings.ToUpper(strings.TrimSpace(req.ReplacementCourseCode))
	scopeCohort := strings.TrimSpace(req.ScopeCohort)
	minC := int64(req.MinCohort)
	maxC := int64(req.MaxCohort)
	if minC == 0 && maxC == 0 {
		minC, maxC = repositories.ParseCohortRange(scopeCohort)
	} else if maxC == 0 {
		maxC = 999
	}
	if scopeCohort == "" {
		if minC <= 1 && maxC >= 999 {
			scopeCohort = "ALL"
		} else if minC > 0 && maxC >= 999 {
			scopeCohort = fmt.Sprintf("K%d-K39", minC)
		} else if minC > 0 && maxC > 0 {
			if minC == maxC {
				scopeCohort = fmt.Sprintf("K%d", minC)
			} else {
				scopeCohort = fmt.Sprintf("K%d-K%d", minC, maxC)
			}
		}
	}

	isBi := int64(0)
	if req.IsBidirectional {
		isBi = 1
	}
	isTr := int64(0)
	if req.IsTransitive {
		isTr = 1
	}

	params := sqlc.UpsertCourseSubstitutionParams{
		ID:                    id,
		OriginalCourseCode:    origCode,
		OriginalCourseName:    strings.TrimSpace(req.OriginalCourseName),
		ReplacementCourseCode: repCode,
		ReplacementCourseName: strings.TrimSpace(req.ReplacementCourseName),
		ScopeCohort:           scopeCohort,
		ScopeMajor:            req.ScopeMajor,
		MinCohort:             minC,
		MaxCohort:             maxC,
		IsBidirectional:       isBi,
		IsTransitive:          isTr,
		CreditsOriginal:       req.CreditsOriginal,
		CreditsReplacement:    req.CreditsReplacement,
		RuleType:              req.RuleType,
		RuleNote:              req.RuleNote,
		SourceReference:       req.SourceReference,
	}
	sub, err := s.repo.UpsertCourseSubstitution(ctx, params)
	if err != nil {
		return nil, err
	}
	return sub.ToResponse(), nil
}

func (s *curriculumService) DeleteSubstitution(ctx context.Context, id string) error {
	return s.repo.DeleteCourseSubstitution(ctx, id)
}

func (s *curriculumService) GetStats(ctx context.Context) (*response.CurriculumStatsResponse, error) {
	currCount, err := s.repo.CountCurriculums(ctx)
	if err != nil {
		return nil, err
	}
	courseCount, err := s.repo.CountCurriculumCourses(ctx)
	if err != nil {
		return nil, err
	}
	subCount, err := s.repo.CountSubstitutions(ctx)
	if err != nil {
		return nil, err
	}
	return &response.CurriculumStatsResponse{
		Curriculums:   currCount,
		Courses:       courseCount,
		Substitutions: subCount,
	}, nil
}

func (s *curriculumService) GetVisualizationData(ctx context.Context, cohort, majorName string) (*response.CurriculumVisualizationDataResponse, error) {
	return s.buildVisualizationData(ctx, cohort, majorName)
}

func (s *curriculumService) buildVisualizationData(ctx context.Context, cohort, majorName string) (*response.CurriculumVisualizationDataResponse, error) {
	curr, courses, err := s.GetCurriculum(ctx, cohort, majorName)
	if err != nil {
		return nil, err
	}

	semesterMap := make(map[int]*response.SemesterGroupResponse)
	blockMap := make(map[string]*response.KnowledgeBlockDistributionResponse)
	var totalCredits float64

	courseMap := make(map[string]*response.CurriculumCourseResponse, len(courses))
	for _, c := range courses {
		courseMap[strings.ToUpper(strings.TrimSpace(c.CourseCode))] = c
		sem := c.SuggestedSemester
		if sem <= 0 {
			sem = 1
		}
		if _, ok := semesterMap[sem]; !ok {
			semesterMap[sem] = &response.SemesterGroupResponse{
				Semester: sem,
				Courses:  make([]*response.CurriculumCourseResponse, 0),
			}
		}
		semesterMap[sem].Courses = append(semesterMap[sem].Courses, c)
		semesterMap[sem].TotalCredits += c.Credits
		semesterMap[sem].CourseCount++

		bName := c.KnowledgeBlock
		if bName == "" {
			bName = "Other"
		}
		if _, ok := blockMap[bName]; !ok {
			blockMap[bName] = &response.KnowledgeBlockDistributionResponse{
				Name: bName,
			}
		}
		blockMap[bName].CourseCount++
		blockMap[bName].TotalCredits += c.Credits
		totalCredits += c.Credits
	}

	semesters := make([]*response.SemesterGroupResponse, 0, len(semesterMap))
	for _, sg := range semesterMap {
		semesters = append(semesters, sg)
	}
	sort.Slice(semesters, func(i, j int) bool {
		return semesters[i].Semester < semesters[j].Semester
	})

	blocks := make([]*response.KnowledgeBlockDistributionResponse, 0, len(blockMap))
	for _, b := range blockMap {
		if totalCredits > 0 {
			b.Percentage = math.Round((b.TotalCredits/totalCredits)*1000) / 10
		}
		b.KnowledgeBlock = b.Name
		blocks = append(blocks, b)
	}
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].TotalCredits > blocks[j].TotalCredits
	})

	prereqRows, err := s.repo.ListAllPrerequisitesForCohortAndMajor(ctx, curr.Cohort, curr.MajorName)
	var edges []*response.PrerequisiteEdgeResponse
	if err == nil {
		edges = make([]*response.PrerequisiteEdgeResponse, len(prereqRows))
		for i, pr := range prereqRows {
			fromCode := strings.ToUpper(strings.TrimSpace(pr.PrerequisiteCourseCode))
			toCode := strings.ToUpper(strings.TrimSpace(pr.CourseCode))
			edge := &response.PrerequisiteEdgeResponse{
				From:           pr.PrerequisiteCourseCode,
				To:             pr.CourseCode,
				FromCourseCode: pr.PrerequisiteCourseCode,
				ToCourseCode:   pr.CourseCode,
				MinGrade:       pr.MinGrade,
				ConditionType:  pr.ConditionType,
			}
			if fc, exists := courseMap[fromCode]; exists {
				edge.FromCourseName = fc.CourseName
				edge.FromCredits = fc.Credits
				edge.FromKnowledgeBlock = fc.KnowledgeBlock
			} else if meta, mErr := s.repo.FindCourseMetadataByCode(ctx, fromCode); mErr == nil && meta != nil {
				edge.FromCourseName = meta.CourseName
				edge.FromCredits = meta.Credits
				edge.FromKnowledgeBlock = meta.KnowledgeBlock
			}
			if tc, exists := courseMap[toCode]; exists {
				edge.ToCourseName = tc.CourseName
				edge.ToCredits = tc.Credits
				edge.ToKnowledgeBlock = tc.KnowledgeBlock
			} else if meta, mErr := s.repo.FindCourseMetadataByCode(ctx, toCode); mErr == nil && meta != nil {
				edge.ToCourseName = meta.CourseName
				edge.ToCredits = meta.Credits
				edge.ToKnowledgeBlock = meta.KnowledgeBlock
			}
			edges[i] = edge
		}
	}

	return &response.CurriculumVisualizationDataResponse{
		Curriculum:        curr,
		TotalCourses:      len(courses),
		TotalCredits:      totalCredits,
		Courses:           courses,
		Semesters:         semesters,
		KnowledgeBlocks:   blocks,
		PrerequisiteEdges: edges,
	}, nil
}

// GetStudentPersonalRoadmap compiles the personal study roadmap by mapping student grades against curriculum requirements.
func (s *curriculumService) GetStudentPersonalRoadmap(ctx context.Context, userID, cohort, majorName string, grades []*response.StudentCourseGradeItem, optimizer RoadmapOptimizer) (*response.StudentPersonalRoadmapDataResponse, error) {
	visData, err := s.buildVisualizationData(ctx, cohort, majorName)
	if err != nil {
		return nil, err
	}

	gradeMap := make(map[string]*response.StudentCourseGradeItem)
	passedSet := make(map[string]bool)
	var completedCredits float64
	var attemptedCredits float64
	var weightedScore10 float64
	var weightedScore4 float64
	var gpaCredits float64
	passedCount := 0
	failedCount := 0

	for _, g := range grades {
		upper := strings.ToUpper(strings.TrimSpace(g.CourseCode))
		gradeMap[upper] = g
		attemptedCredits += g.Credits
		if g.IsPassed && g.GradeLetter != "F" {
			passedSet[upper] = true
			completedCredits += g.Credits
			passedCount++
		} else {
			failedCount++
		}
		if g.Score10 > 0 && g.Credits > 0 {
			weightedScore10 += g.Score10 * g.Credits
			weightedScore4 += g.Score4 * g.Credits
			gpaCredits += g.Credits
		}
	}

	var cpa10 float64
	var cpa4 float64
	if gpaCredits > 0 {
		cpa10 = weightedScore10 / gpaCredits
		cpa4 = weightedScore4 / gpaCredits
	}

	prereqMap := make(map[string][]string)
	for _, edge := range visData.PrerequisiteEdges {
		toUpper := strings.ToUpper(strings.TrimSpace(edge.To))
		fromUpper := strings.ToUpper(strings.TrimSpace(edge.From))
		prereqMap[toUpper] = append(prereqMap[toUpper], fromUpper)
	}

	var criticalPathMap map[string]bool
	var optResult *response.RoadmapOptimizationResultResponse
	if optimizer != nil {
		transcriptList := make([]models.StudentCourseGrade, 0, len(grades))
		for _, g := range grades {
			transcriptList = append(transcriptList, models.StudentCourseGrade{
				CourseCode: g.CourseCode,
				CourseName: g.CourseName,
				Credits:    g.Credits,
				Grade:      g.GradeLetter,
				Score:      g.Score10,
				IsRetake:   !g.IsPassed || g.GradeLetter == "F",
			})
		}
		curSem := 1
		if completedCredits >= 105 {
			curSem = 7
		} else if completedCredits >= 90 {
			curSem = 6
		} else if completedCredits >= 70 {
			curSem = 5
		} else if completedCredits >= 50 {
			curSem = 4
		} else if completedCredits >= 30 {
			curSem = 3
		} else if completedCredits >= 15 {
			curSem = 2
		}
		res, err := optimizer.OptimizeRoadmap(ctx, &models.RoadmapOptimizationRequest{
			Cohort:               visData.Curriculum.Cohort,
			MajorName:            visData.Curriculum.MajorName,
			CurrentSemester:      curSem,
			TargetGraduationTerm: 8,
			Transcript:           transcriptList,
		})
		if err == nil && res != nil {
			optResult = res
			criticalPathMap = make(map[string]bool)
			for _, cp := range res.CriticalPathCourses {
				criticalPathMap[strings.ToUpper(strings.TrimSpace(cp))] = true
			}
		}
	}

	studentSemesters := make([]*response.StudentSemesterGroupResponse, 0, len(visData.Semesters))
	for _, sg := range visData.Semesters {
		stGroup := &response.StudentSemesterGroupResponse{
			Semester:     sg.Semester,
			TotalCredits: sg.TotalCredits,
			CourseCount:  sg.CourseCount,
			Courses:      make([]*response.StudentRoadmapCourseResponse, 0, len(sg.Courses)),
		}
		for _, c := range sg.Courses {
			upperCode := strings.ToUpper(strings.TrimSpace(c.CourseCode))
			stCourse := &response.StudentRoadmapCourseResponse{
				CurriculumCourseResponse: *c,
				IsCriticalPath:           criticalPathMap[upperCode],
			}
			if g, exists := gradeMap[upperCode]; exists {
				stCourse.Score10 = g.Score10
				stCourse.Score4 = g.Score4
				stCourse.GradeLetter = g.GradeLetter
				stCourse.TermTaken = g.TermTaken
				if g.IsPassed && g.GradeLetter != "F" {
					stCourse.Status = "PASSED"
				} else {
					stCourse.Status = "FAILED_RETAKE"
				}
			} else {
				prereqs := prereqMap[upperCode]
				var missing []string
				for _, p := range prereqs {
					if !passedSet[p] {
						missing = append(missing, p)
					}
				}
				stCourse.MissingPrereqs = missing
				if len(missing) == 0 {
					stCourse.Status = "UNLOCKED"
				} else {
					stCourse.Status = "LOCKED"
				}
			}
			stGroup.Courses = append(stGroup.Courses, stCourse)
		}
		studentSemesters = append(studentSemesters, stGroup)
	}

	remainingCredits := visData.TotalCredits - completedCredits
	if remainingCredits < 0 {
		remainingCredits = 0
	}
	var progressPct float64
	if visData.TotalCredits > 0 {
		progressPct = (completedCredits / visData.TotalCredits) * 100
		if progressPct > 100 {
			progressPct = 100
		}
	}

	summary := &response.StudentRoadmapSummaryResponse{
		Cohort:                 visData.Curriculum.Cohort,
		MajorName:              visData.Curriculum.MajorName,
		TotalCurriculumCredits: visData.TotalCredits,
		TotalCreditsCompleted:  completedCredits,
		TotalCreditsRemaining:  remainingCredits,
		ProgressPercentage:     progressPct,
		CumulativeGPA10:        cpa10,
		CumulativeGPA4:         cpa4,
		TotalCourses:           visData.TotalCourses,
		PassedCoursesCount:     passedCount,
		FailedCoursesCount:     failedCount,
	}

	return &response.StudentPersonalRoadmapDataResponse{
		Summary:           summary,
		Semesters:         studentSemesters,
		KnowledgeBlocks:   visData.KnowledgeBlocks,
		PrerequisiteEdges: visData.PrerequisiteEdges,
		OptimizationPlan:  optResult,
	}, nil
}

// FindMajorInText identifies the academic major from freeform text using database metadata.
func (s *curriculumService) FindMajorInText(ctx context.Context, text string) string {
	if text == "" {
		return ""
	}
	majors, err := s.repo.ListAllMajorMetadata(ctx)
	if err != nil || len(majors) == 0 {
		return ""
	}
	normQuery := guardrail.FoldToASCII(strings.ToLower(text))
	var candidates []sqlc.ListAllMajorMetadataRow
	candidates = append(candidates, majors...)
	sort.Slice(candidates, func(i, j int) bool {
		return len(candidates[i].MajorName) > len(candidates[j].MajorName)
	})
	for _, m := range candidates {
		normMajor := guardrail.FoldToASCII(strings.ToLower(m.MajorName))
		if normMajor != "" && isWordBoundaryMatch(normQuery, normMajor) {
			return m.MajorName
		}
		if m.Aliases != "" {
			for _, a := range strings.Split(m.Aliases, ",") {
				normAlias := guardrail.FoldToASCII(strings.TrimSpace(strings.ToLower(a)))
				if normAlias != "" && isWordBoundaryMatch(normQuery, normAlias) {
					return m.MajorName
				}
			}
		}
	}
	return ""
}

func isWordBoundaryMatch(text, pattern string) bool {
	if pattern == "" || text == "" {
		return false
	}
	idx := strings.Index(text, pattern)
	for idx >= 0 {
		startOk := idx == 0 || !isAlphaNumByte(text[idx-1])
		endIdx := idx + len(pattern)
		endOk := endIdx == len(text) || !isAlphaNumByte(text[endIdx])
		if startOk && endOk {
			return true
		}
		next := strings.Index(text[idx+1:], pattern)
		if next < 0 {
			break
		}
		idx += 1 + next
	}
	return false
}

func isAlphaNumByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

