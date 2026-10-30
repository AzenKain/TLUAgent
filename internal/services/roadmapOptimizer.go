package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/apperrors"
)

// RoadmapOptimizer orchestrates DAG-based curriculum progression, rule waterfalls, and pacing optimization.
type RoadmapOptimizer interface {
	OptimizeRoadmap(ctx context.Context, req *models.RoadmapOptimizationRequest) (*response.RoadmapOptimizationResultResponse, error)
	EvaluateDiagnosticQuiz(ctx context.Context, answers *models.PersonalityQuizAnswers) (*response.PersonalityQuizResultResponse, error)
}

type roadmapOptimizer struct {
	curriculumRepo repositories.CurriculumRepository
	memoryRepo     repositories.MemoryRepository
}

// NewRoadmapOptimizer creates a new instance of RoadmapOptimizer.
func NewRoadmapOptimizer(curriculumRepo repositories.CurriculumRepository, memoryRepo ...repositories.MemoryRepository) RoadmapOptimizer {
	o := &roadmapOptimizer{curriculumRepo: curriculumRepo}
	if len(memoryRepo) > 0 {
		o.memoryRepo = memoryRepo[0]
	}
	return o
}

type dagNode struct {
	course          *models.CurriculumCourse
	prereqs         []string
	dependents      []string
	substitutions   []string
	outDegree       int
	totalDownstream int
	longestPath     int
	isFailedRetake  bool
	priorityScore   float64
}

// EvaluateDiagnosticQuiz processes diagnostic quiz answers into pacing targets and competency scores.
func (s *roadmapOptimizer) EvaluateDiagnosticQuiz(ctx context.Context, answers *models.PersonalityQuizAnswers) (*response.PersonalityQuizResultResponse, error) {
	if answers == nil {
		return (&models.PersonalityQuizResult{
			TargetCreditsPerSemester:  17.0,
			RecommendedGraduationTerm: 8,
			CompetencyRadar: map[string]float64{
				"programming": 75.0,
				"mathematics": 75.0,
				"networking":  75.0,
				"softskills":  75.0,
			},
			DeliveryBias: "balanced",
			SummaryNotes: "Standard 4-year curriculum (8 semesters) with average load of 16-18 credits per semester.",
		}).ToResponse(), nil
	}

	targetCredits := 17.0
	recTerm := 8
	switch strings.ToLower(strings.TrimSpace(answers.PacePreference)) {
	case "fast":
		targetCredits = 19.0
		recTerm = 7
	case "relaxed":
		targetCredits = 14.5
		recTerm = 9
	default:
		targetCredits = 17.0
		recTerm = 8
	}

	radar := map[string]float64{
		"programming": 70.0,
		"mathematics": 70.0,
		"networking":  70.0,
		"softskills":  70.0,
	}

	switch strings.ToLower(strings.TrimSpace(answers.PrimaryStrength)) {
	case "programming":
		radar["programming"] = 90.0
		radar["networking"] = 75.0
		radar["mathematics"] = 70.0
		radar["softskills"] = 65.0
	case "mathematics":
		radar["mathematics"] = 95.0
		radar["programming"] = 80.0
		radar["networking"] = 70.0
		radar["softskills"] = 60.0
	case "networking":
		radar["networking"] = 90.0
		radar["programming"] = 75.0
		radar["mathematics"] = 70.0
		radar["softskills"] = 70.0
	case "softskills":
		radar["softskills"] = 90.0
		radar["programming"] = 65.0
		radar["mathematics"] = 65.0
		radar["networking"] = 70.0
	}

	delivery := "balanced"
	switch strings.ToLower(strings.TrimSpace(answers.DeliveryPreference)) {
	case "elearning":
		delivery = "elearning"
	case "in_person":
		delivery = "in_person"
	default:
		delivery = "balanced"
	}

	notes := fmt.Sprintf("Target pace: %.1f credits/term, estimated completion in %d semesters.", targetCredits, recTerm)
	if answers.SummerWillingness {
		notes += " Ready for summer semester to reduce regular term load."
	}

	return (&models.PersonalityQuizResult{
		TargetCreditsPerSemester:  targetCredits,
		RecommendedGraduationTerm: recTerm,
		CompetencyRadar:           radar,
		DeliveryBias:              delivery,
		SummaryNotes:              notes,
	}).ToResponse(), nil
}

// OptimizeRoadmap executes the topological scheduling algorithm with academic constraints.
func (s *roadmapOptimizer) OptimizeRoadmap(ctx context.Context, req *models.RoadmapOptimizationRequest) (*response.RoadmapOptimizationResultResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("optimization request cannot be nil")
	}

	cohort := strings.ToUpper(strings.TrimSpace(req.Cohort))
	if cohort == "" {
		return nil, apperrors.New(apperrors.ErrMissingCohort, "cohort is required to optimize a study roadmap")
	}

	majorName := strings.TrimSpace(req.MajorName)
	resolved := resolveMajorMetadata(ctx, s.curriculumRepo, cohort, majorName)
	if resolved.MajorName != "" {
		majorName = resolved.MajorName
	}

	courses, err := s.curriculumRepo.GetCurriculumCoursesByCohortAndMajor(ctx, cohort, majorName)
	if err != nil {
		return nil, fmt.Errorf("fetch curriculum courses: %w", err)
	}
	if len(courses) == 0 {
		return nil, fmt.Errorf("no curriculum courses found for cohort %s and major %s", cohort, majorName)
	}

	prereqEdges, err := s.curriculumRepo.ListAllPrerequisitesForCohortAndMajor(ctx, cohort, majorName)
	if err != nil {
		return nil, fmt.Errorf("fetch curriculum prerequisites: %w", err)
	}

	cohortNum := parseCohortNumber(cohort)
	var subs []*models.CourseSubstitution
	if sList, err := s.curriculumRepo.ListSubstitutionsByCohortAndMajor(ctx, cohortNum, resolved.MajorCode, resolved.FacultyCode); err == nil {
		subs = append(subs, sList...)
	}

	distinctMajors, _ := s.curriculumRepo.ListDistinctSubstitutionMajors(ctx, cohortNum)
	for _, sm := range distinctMajors {
		if sm != resolved.MajorCode && sm != resolved.FacultyCode {
			if dList, dErr := s.curriculumRepo.ListSubstitutionsByCohortAndMajor(ctx, cohortNum, sm, resolved.FacultyCode); dErr == nil {
				subs = append(subs, dList...)
			}
		}
	}

	substitutesFor := make(map[string][]string)
	seenSub := make(map[string]bool)
	for _, sub := range subs {
		orig := strings.ToUpper(strings.TrimSpace(sub.OriginalCourseCode))
		repl := strings.ToUpper(strings.TrimSpace(sub.ReplacementCourseCode))
		if orig == "" || repl == "" {
			continue
		}
		pairKey := orig + "->" + repl
		if !seenSub[pairKey] {
			seenSub[pairKey] = true
			substitutesFor[orig] = append(substitutesFor[orig], repl)
		}
		if sub.IsBidirectional {
			revKey := repl + "->" + orig
			if !seenSub[revKey] {
				seenSub[revKey] = true
				substitutesFor[repl] = append(substitutesFor[repl], orig)
			}
		}
	}

	var totalCurrCredits float64
	currCourseMap := make(map[string]*models.CurriculumCourse, len(courses))
	for _, c := range courses {
		currCourseMap[c.CourseCode] = c
		totalCurrCredits += c.Credits
	}

	passedMap := make(map[string]bool)
	failedMap := make(map[string]bool)
	var accumulatedCredits float64
	var retakeCredits float64
	var crossMajorCourses []string
	var extraLanguages []string

	for _, g := range req.Transcript {
		code := strings.ToUpper(strings.TrimSpace(g.CourseCode))
		if code == "" {
			continue
		}

		isPassed := g.Score >= 4.0 || (g.Grade != "" && !strings.EqualFold(g.Grade, "F"))
		if isPassed {
			passedMap[code] = true
			if c, exists := currCourseMap[code]; exists {
				accumulatedCredits += c.Credits
			} else {
				crossMajorCourses = append(crossMajorCourses, code)
				if g.Credits > 0 {
					accumulatedCredits += g.Credits
				}
			}
		} else {
			failedMap[code] = true
			if c, exists := currCourseMap[code]; exists {
				retakeCredits += c.Credits
			}
		}

		if g.IsRetake {
			if c, exists := currCourseMap[code]; exists {
				retakeCredits += c.Credits
			}
		}

		if isForeignLanguageCode(code) {
			extraLanguages = append(extraLanguages, code)
		}
	}

	for _, c := range courses {
		if !passedMap[c.CourseCode] {
			for _, repl := range substitutesFor[c.CourseCode] {
				if passedMap[repl] {
					passedMap[c.CourseCode] = true
					if _, wasInCurr := currCourseMap[repl]; !wasInCurr {
						if getTranscriptCredits(req.Transcript, repl) <= 0 {
							accumulatedCredits += c.Credits
						}
					}
					filteredCross := make([]string, 0, len(crossMajorCourses))
					for _, cm := range crossMajorCourses {
						if cm != repl {
							filteredCross = append(filteredCross, cm)
						}
					}
					crossMajorCourses = filteredCross
					break
				}
			}
		}
	}

	waterfall := s.calculateFreeElectivesWaterfall(passedMap, crossMajorCourses, extraLanguages)

	standing := s.resolveAcademicStanding(ctx, req)
	maxRetakeAllowed := totalCurrCredits * 0.05
	downgradeWarning := models.DegreeDowngradeRiskWarning{
		TotalRetakeCredits: retakeCredits,
		MaxAllowedCredits:  maxRetakeAllowed,
	}
	if isHonorsStanding(standing) {
		downgradeWarning.IsAtRisk = retakeCredits > maxRetakeAllowed
		if downgradeWarning.IsAtRisk {
			downgradeWarning.WarningMessage = fmt.Sprintf(
				"Warning: Accumulated %.1f retake credits exceeds 5%% curriculum threshold (max %.1f credits). Honors classifications may be downgraded by one rank upon graduation.",
				retakeCredits, maxRetakeAllowed,
			)
		}
	} else {
		downgradeWarning.Note = fmt.Sprintf(
			"Ngưỡng hạ bậc bằng tốt nghiệp 5%% (%.1f tín chỉ) chỉ áp dụng cho học lực Xuất sắc/Giỏi; học lực hiện tại (%s) không bị xét hạ bậc vì số tín chỉ học lại.",
			maxRetakeAllowed, displayStanding(standing),
		)
	}

	var remainingCourses []*models.CurriculumCourse
	for _, c := range courses {
		if !passedMap[c.CourseCode] {
			remainingCourses = append(remainingCourses, c)
		}
	}

	if len(remainingCourses) == 0 {
		return (&models.RoadmapOptimizationResult{
			Cohort:                 cohort,
			MajorName:              majorName,
			AccumulatedCredits:     accumulatedCredits,
			RemainingCredits:       0,
			TotalCurriculumCredits: totalCurrCredits,
			CurrentSemester:        req.CurrentSemester,
			TargetGraduationTerm:   req.TargetGraduationTerm,
			Semesters:              []*models.ScheduledSemester{},
			CriticalPathCourses:    []string{},
			FreeElectives:          waterfall,
			DowngradeRisk:          downgradeWarning,
			Recommendations:        []string{"Congratulations! You have completed all curriculum requirements."},
		}).ToResponse(), nil
	}

	dag := s.buildDAG(remainingCourses, prereqEdges, failedMap, substitutesFor)

	targetPace := 17.0
	targetTerm := req.TargetGraduationTerm
	if targetTerm <= 0 {
		targetTerm = 8
	}

	var quizResult *response.PersonalityQuizResultResponse
	if req.QuizAnswers != nil {
		qr, qErr := s.EvaluateDiagnosticQuiz(ctx, req.QuizAnswers)
		if qErr == nil && qr != nil {
			quizResult = qr
			targetPace = qr.TargetCreditsPerSemester
			if req.TargetGraduationTerm <= 0 {
				targetTerm = qr.RecommendedGraduationTerm
			}
		}
	}

	startSem := req.CurrentSemester + 1
	if startSem <= 1 {
		startSem = 1
	}

	scheduledSemesters, criticalPaths := s.scheduleSemesters(
		dag,
		startSem,
		targetTerm,
		targetPace,
		passedMap,
		failedMap,
		req.QuizAnswers,
	)

	var remainingCredits float64
	for _, c := range remainingCourses {
		remainingCredits += c.Credits
	}

	recommendations := s.generateRecommendations(
		downgradeWarning,
		waterfall,
		criticalPaths,
		targetTerm,
		startSem,
		quizResult,
	)

	return (&models.RoadmapOptimizationResult{
		Cohort:                 cohort,
		MajorName:              majorName,
		AccumulatedCredits:     accumulatedCredits,
		RemainingCredits:       remainingCredits,
		TotalCurriculumCredits: totalCurrCredits,
		CurrentSemester:        req.CurrentSemester,
		TargetGraduationTerm:   targetTerm,
		Semesters:              scheduledSemesters,
		CriticalPathCourses:    criticalPaths,
		FreeElectives:          waterfall,
		DowngradeRisk:          downgradeWarning,
		Recommendations:        recommendations,
	}).ToResponse(), nil
}

func (s *roadmapOptimizer) calculateFreeElectivesWaterfall(passedMap map[string]bool, crossMajor []string, languages []string) models.FreeElectiveWaterfall {
	target := 5.0
	var earned float64
	var cross []string
	var extraLang []string

	for _, code := range crossMajor {
		cross = append(cross, code)
		earned += 3.0
	}

	langPairs := make(map[string]int)
	for _, l := range languages {
		prefix := extractLanguagePrefix(l)
		if prefix != "" {
			langPairs[prefix]++
		}
	}

	for prefix, count := range langPairs {
		if count > 2 {
			surplus := count - 2
			extraLang = append(extraLang, fmt.Sprintf("%s (%d HP dư)", prefix, surplus))
			earned += float64(surplus) * 3.0
		}
	}

	return models.FreeElectiveWaterfall{
		TargetCredits:     target,
		EarnedCredits:     math.Min(earned, target),
		CrossMajorCourses: cross,
		ExtraLanguages:    extraLang,
		IsFulfilled:       earned >= target,
	}
}

func (s *roadmapOptimizer) buildDAG(
	remaining []*models.CurriculumCourse,
	prereqEdges []*models.CoursePrerequisite,
	failedMap map[string]bool,
	substitutesFor map[string][]string,
) map[string]*dagNode {
	nodes := make(map[string]*dagNode, len(remaining))
	for _, c := range remaining {
		nodes[c.CourseCode] = &dagNode{
			course:         c,
			prereqs:        make([]string, 0),
			dependents:     make([]string, 0),
			substitutions:  substitutesFor[c.CourseCode],
			isFailedRetake: failedMap[c.CourseCode],
		}
	}

	for _, edge := range prereqEdges {
		childCode := strings.ToUpper(edge.CourseCode)
		parentCode := strings.ToUpper(edge.PrerequisiteCourseCode)

		childNode, childExists := nodes[childCode]
		parentNode, parentExists := nodes[parentCode]

		if childExists && parentExists {
			childNode.prereqs = append(childNode.prereqs, parentCode)
			parentNode.dependents = append(parentNode.dependents, childCode)
		}
	}

	for _, node := range nodes {
		node.outDegree = len(node.dependents)
		node.totalDownstream = countTransitiveDownstream(node.course.CourseCode, nodes)
		node.longestPath = computeLongestPath(node.course.CourseCode, nodes, make(map[string]int))
	}

	for _, node := range nodes {
		score := float64(node.totalDownstream*50 + node.longestPath*30 + node.outDegree*20)
		if node.isFailedRetake {
			score += 1000.0
		}
		if isDigitalCitizenCourse(node.course.CourseCode) {
			score += 350.0
		}
		if isEnglishCourse(node.course.CourseCode) {
			score += 250.0
		}
		if node.course.IsElearning {
			score -= 50.0
		}
		node.priorityScore = score
	}

	return nodes
}

func countTransitiveDownstream(start string, nodes map[string]*dagNode) int {
	visited := make(map[string]bool)
	queue := []string{start}
	visited[start] = true

	count := 0
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		node, exists := nodes[curr]
		if !exists {
			continue
		}

		for _, dep := range node.dependents {
			if !visited[dep] {
				visited[dep] = true
				count++
				queue = append(queue, dep)
			}
		}
	}
	return count
}

func computeLongestPath(curr string, nodes map[string]*dagNode, memo map[string]int) int {
	if val, ok := memo[curr]; ok {
		return val
	}

	node, exists := nodes[curr]
	if !exists || len(node.dependents) == 0 {
		memo[curr] = 1
		return 1
	}

	maxChild := 0
	for _, dep := range node.dependents {
		childLen := computeLongestPath(dep, nodes, memo)
		if childLen > maxChild {
			maxChild = childLen
		}
	}

	res := maxChild + 1
	memo[curr] = res
	return res
}

func (s *roadmapOptimizer) scheduleSemesters(
	dag map[string]*dagNode,
	startSem int,
	targetTerm int,
	targetPace float64,
	initiallyPassed map[string]bool,
	failedMap map[string]bool,
	quiz *models.PersonalityQuizAnswers,
) ([]*models.ScheduledSemester, []string) {
	completed := make(map[string]bool)
	for k, v := range initiallyPassed {
		completed[k] = v
	}

	unassigned := make(map[string]*dagNode, len(dag))
	for k, v := range dag {
		unassigned[k] = v
	}

	var scheduledSemesters []*models.ScheduledSemester
	currentSemNum := startSem
	maxSemesters := 14

	criticalMap := make(map[string]bool)
	for _, node := range dag {
		if node.totalDownstream >= 3 || node.longestPath >= 3 {
			criticalMap[node.course.CourseCode] = true
		}
	}

	militaryScheduled := completed["PG121"]
	allowSummer := quiz != nil && quiz.SummerWillingness

	for len(unassigned) > 0 && currentSemNum <= maxSemesters {
		isSummer := allowSummer && (currentSemNum%2 == 1 && currentSemNum > startSem)

		maxCreditsThisSem := targetPace + 2.0
		minCreditsThisSem := targetPace - 2.0
		if minCreditsThisSem < 12.0 {
			minCreditsThisSem = 12.0
		}
		if isSummer {
			maxCreditsThisSem = 6.0
			minCreditsThisSem = 3.0
		}

		eligible := make([]*dagNode, 0)
		for _, node := range unassigned {
			if isMilitaryCourse(node.course.CourseCode) {
				continue
			}

			prereqsSatisfied := true
			for _, p := range node.prereqs {
				if !completed[p] {
					prereqsSatisfied = false
					break
				}
			}
			if prereqsSatisfied {
				eligible = append(eligible, node)
			}
		}

		if len(eligible) == 0 {
			currentSemNum++
			continue
		}

		sort.Slice(eligible, func(i, j int) bool {
			return eligible[i].priorityScore > eligible[j].priorityScore
		})

		var semCourses []*models.ScheduledCourse
		var currentSemCredits float64
		hasEnglishThisSem := false
		scheduledInThisSem := make([]*dagNode, 0)

		for _, node := range eligible {
			if currentSemCredits+node.course.Credits > maxCreditsThisSem && len(semCourses) > 0 {
				continue
			}

			if isEnglishCourse(node.course.CourseCode) {
				if hasEnglishThisSem {
					continue
				}
				hasEnglishThisSem = true
			}

			reason := s.determinePriorityReason(node)
			semCourses = append(semCourses, &models.ScheduledCourse{
				CourseCode:           node.course.CourseCode,
				CourseName:           node.course.CourseName,
				Credits:              node.course.Credits,
				KnowledgeBlock:       node.course.KnowledgeBlock,
				IsElearning:          node.course.IsElearning,
				IsRetake:             node.isFailedRetake,
				PriorityReason:       reason,
				PrerequisitesMet:     node.prereqs,
				AvailableSubstitutes: node.substitutions,
			})

			currentSemCredits += node.course.Credits
			scheduledInThisSem = append(scheduledInThisSem, node)
		}

		if currentSemCredits < minCreditsThisSem && len(unassigned) > len(scheduledInThisSem) {
			for _, node := range eligible {
				if !node.course.IsElearning {
					continue
				}
				alreadyScheduled := false
				for _, sc := range scheduledInThisSem {
					if sc.course.CourseCode == node.course.CourseCode {
						alreadyScheduled = true
						break
					}
				}
				if !alreadyScheduled && currentSemCredits+node.course.Credits <= maxCreditsThisSem+2.0 {
					semCourses = append(semCourses, &models.ScheduledCourse{
						CourseCode:           node.course.CourseCode,
						CourseName:           node.course.CourseName,
						Credits:              node.course.Credits,
						KnowledgeBlock:       node.course.KnowledgeBlock,
						IsElearning:          true,
						IsRetake:             node.isFailedRetake,
						PriorityReason:       "E-learning buffer to fulfill minimum credit requirement without scheduling conflicts.",
						PrerequisitesMet:     node.prereqs,
						AvailableSubstitutes: node.substitutions,
					})
					currentSemCredits += node.course.Credits
					scheduledInThisSem = append(scheduledInThisSem, node)
				}
			}
		}

		for _, node := range scheduledInThisSem {
			delete(unassigned, node.course.CourseCode)
			completed[node.course.CourseCode] = true
		}

		feedback := fmt.Sprintf("Semester %d: %.1f credits (%d courses).", currentSemNum, currentSemCredits, len(semCourses))
		scheduledSemesters = append(scheduledSemesters, &models.ScheduledSemester{
			SemesterNumber: currentSemNum,
			IsSummer:       isSummer,
			TotalCredits:   currentSemCredits,
			Courses:        semCourses,
			PaceFeedback:   feedback,
		})

		if !militaryScheduled && currentSemNum >= 2 {
			if node, exists := unassigned["PG121"]; exists {
				delete(unassigned, "PG121")
				completed["PG121"] = true
				militaryScheduled = true

				scheduledSemesters = append(scheduledSemesters, &models.ScheduledSemester{
					SemesterNumber: currentSemNum,
					IsSummer:       true,
					TotalCredits:   node.course.Credits,
					Courses: []*models.ScheduledCourse{
						{
							CourseCode:       node.course.CourseCode,
							CourseName:       node.course.CourseName,
							Credits:          node.course.Credits,
							KnowledgeBlock:   node.course.KnowledgeBlock,
							IsElearning:      false,
							IsRetake:         false,
							PriorityReason:   "Military training organized intensively during whole-cohort summer term.",
							PrerequisitesMet: []string{},
						},
					},
					PaceFeedback: "Intensive military summer term according to university schedule.",
				})
			}
		}

		currentSemNum++
	}

	criticalList := make([]string, 0, len(criticalMap))
	for code := range criticalMap {
		criticalList = append(criticalList, code)
	}
	sort.Strings(criticalList)

	return scheduledSemesters, criticalList
}

func (s *roadmapOptimizer) determinePriorityReason(node *dagNode) string {
	if node.isFailedRetake {
		return "Failed in previous semester, required retake to unblock academic progress."
	}
	if isDigitalCitizenCourse(node.course.CourseCode) {
		return "Equips document formatting, academic writing, and presentation skills early."
	}
	if isEnglishCourse(node.course.CourseCode) {
		return "Sequential English pathway, maintain continuous progression each term."
	}
	if node.totalDownstream >= 4 {
		return fmt.Sprintf("Critical bottleneck course (unlocks %d downstream courses).", node.totalDownstream)
	}
	if node.course.IsElearning {
		return "Flexible online LMS delivery acts as scheduling buffer."
	}
	return "Required course according to standard curriculum progression."
}

func (s *roadmapOptimizer) generateRecommendations(
	downgrade models.DegreeDowngradeRiskWarning,
	waterfall models.FreeElectiveWaterfall,
	criticalPaths []string,
	targetTerm int,
	startSem int,
	quiz *response.PersonalityQuizResultResponse,
) []string {
	var recs []string

	if downgrade.IsAtRisk {
		recs = append(recs, fmt.Sprintf(
			"Critical warning: Accumulated %.1f retake credits. Avoid further retakes to prevent degree classification downgrade.",
			downgrade.TotalRetakeCredits,
		))
	} else if downgrade.Note != "" {
		recs = append(recs, downgrade.Note)
	} else if downgrade.TotalRetakeCredits > 0 {
		recs = append(recs, fmt.Sprintf(
			"Retake credit monitoring: Currently accumulated %.1f / %.1f safe retake credits.",
			downgrade.TotalRetakeCredits, downgrade.MaxAllowedCredits,
		))
	}

	if len(criticalPaths) > 0 {
		recs = append(recs, fmt.Sprintf(
			"High priority critical path courses: %s. Failing these will delay graduation timeline.",
			strings.Join(criticalPaths, ", "),
		))
	}

	if !waterfall.IsFulfilled {
		missing := waterfall.TargetCredits - waterfall.EarnedCredits
		recs = append(recs, fmt.Sprintf(
			"Free electives: Need %.1f additional free elective credits (cross-major or additional foreign language courses).",
			missing,
		))
	} else {
		recs = append(recs, "Free electives: Completed all 5.0 required credits via waterfall allocation.")
	}

	if quiz != nil {
		recs = append(recs, quiz.SummaryNotes)
	}

	return recs
}

func isEnglishCourse(code string) bool {
	upper := strings.ToUpper(code)
	return strings.HasPrefix(upper, "GE") || strings.HasPrefix(upper, "ENG")
}

func isMilitaryCourse(code string) bool {
	upper := strings.ToUpper(code)
	return strings.HasPrefix(upper, "PG") || strings.Contains(upper, "GDQP")
}

func isDigitalCitizenCourse(code string) bool {
	upper := strings.ToUpper(code)
	return upper == "CS102" || strings.Contains(upper, "THVP")
}

func isForeignLanguageCode(code string) bool {
	upper := strings.ToUpper(code)
	return strings.HasPrefix(upper, "JA") ||
		strings.HasPrefix(upper, "KO") ||
		strings.HasPrefix(upper, "ZH") ||
		strings.HasPrefix(upper, "FR") ||
		strings.HasPrefix(upper, "RU")
}

func extractLanguagePrefix(code string) string {
	upper := strings.ToUpper(code)
	if len(upper) >= 2 {
		return upper[:2]
	}
	return ""
}

func getTranscriptCredits(transcript []models.StudentCourseGrade, code string) float64 {
	for _, g := range transcript {
		if strings.EqualFold(g.CourseCode, code) {
			return g.Credits
		}
	}
	return 0
}

func (s *roadmapOptimizer) resolveAcademicStanding(ctx context.Context, req *models.RoadmapOptimizationRequest) string {
	if standing := strings.ToUpper(strings.TrimSpace(req.AcademicStanding)); standing != "" {
		return standing
	}
	if s.memoryRepo == nil || req.UserID == "" {
		return ""
	}
	profile, err := s.memoryRepo.GetStudentProfile(ctx, req.UserID)
	if err != nil || profile == nil {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(profile.AcademicStanding))
}

func isHonorsStanding(standing string) bool {
	return standing == "XUAT_SAC" || standing == "GIOI"
}

func displayStanding(standing string) string {
	if standing == "" {
		return "chưa xác định"
	}
	return standing
}
