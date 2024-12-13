package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
)

// 简化的课程时间结构
type ClassroomSchedule struct {
	Room          string `json:"room"`          // 教室名称
	FreeSlots     []int  `json:"freeSlots"`     // 空闲节数
	OccupiedSlots []int  `json:"occupiedSlots"` // 占用节数
}

// API 响应结构
type ApiResponse struct {
	Status  int                 `json:"status"`
	Message string              `json:"message"`
	Data    []ClassroomSchedule `json:"data"`
}

// UESTC API 响应结构
type UCourse struct {
	XQJ  string `json:"xqj"`  // 星期几
	KSJC string `json:"ksjc"` // 开始节次
	JSJC string `json:"jsjc"` // 结束节次
}

type UResponse struct {
	Code    int       `json:"code"`
	Message string    `json:"message"`
	Data    []UCourse `json:"data"`
}

// 缓存结构
type Cache struct {
	sync.RWMutex
	data map[string][]UCourse
}

func NewCache() *Cache {
	return &Cache{
		data: make(map[string][]UCourse),
	}
}

var scheduleCache *Cache

func main() {
	// 加载环境变量
	godotenv.Load()

	scheduleCache = NewCache()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   strings.Split(getEnv("ALLOWED_ORIGINS", "http://localhost:3000"), ","),
		AllowedMethods:   []string{"GET", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/api/schedules", handleGetSchedules)

	// 启动缓存刷新协程
	go refreshCacheRoutine()

	port := getEnv("PORT", "8080")
	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func handleGetSchedules(w http.ResponseWriter, r *http.Request) {
	// 获取当前星期几
	weekday := time.Now().Weekday()
	if weekday == time.Sunday {
		weekday = 7
	}
	weekdayStr := strconv.Itoa(int(weekday))
	log.Printf("当前星期: %s", weekdayStr)

	// 获取教室列表
	classrooms := strings.Split(getEnv("CLASSROOMS", "A101,A102,A103"), ",")
	log.Printf("获��到的教室列表: %v", classrooms)

	var schedules []ClassroomSchedule

	for _, room := range classrooms {
		log.Printf("处理教室: %s", room)
		schedule := ClassroomSchedule{
			Room:          room,
			FreeSlots:     make([]int, 0),
			OccupiedSlots: make([]int, 0),
		}

		// 获取课程数据
		courses := getCourseData(room, weekdayStr)
		log.Printf("教室 %s 获取到的课程数据: %+v", room, courses)

		// 计算空闲和占用时段
		occupiedSlots := make(map[int]bool)
		for _, course := range courses {
			start, _ := strconv.Atoi(course.KSJC)
			end, _ := strconv.Atoi(course.JSJC)
			log.Printf("教室 %s 课程时段: %d-%d", room, start, end)
			for i := start; i <= end; i++ {
				occupiedSlots[i] = true
				schedule.OccupiedSlots = append(schedule.OccupiedSlots, i)
			}
		}

		// 计算空闲时段 (假设一天有12节课)
		for i := 1; i <= 12; i++ {
			if !occupiedSlots[i] {
				schedule.FreeSlots = append(schedule.FreeSlots, i)
			}
		}

		log.Printf("教室 %s 最终结果 - 空闲时段: %v, 占用时段: %v",
			room, schedule.FreeSlots, schedule.OccupiedSlots)

		schedules = append(schedules, schedule)
	}

	response := ApiResponse{
		Status:  200,
		Message: "success",
		Data:    schedules,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getCourseData(room string, weekday string) []UCourse {
	cacheKey := fmt.Sprintf("%s-%s", room, weekday)
	log.Printf("获取课程数据 - 教室: %s, 星期: %s", room, weekday)

	// 检查缓存
	scheduleCache.RLock()
	if courses, ok := scheduleCache.data[cacheKey]; ok {
		scheduleCache.RUnlock()
		log.Printf("从缓存获取到数据: %+v", courses)
		return courses
	}
	scheduleCache.RUnlock()

	// 从 UESTC API 获取数据
	courses := fetchFromU(room)
	log.Printf("从 UESTC API 获取到的原始数据: %+v", courses)

	// 过滤当天的课程
	todayCourses := make([]UCourse, 0)
	for _, course := range courses {
		if course.XQJ == weekday {
			todayCourses = append(todayCourses, course)
		}
	}
	log.Printf("过滤后当天的课程: %+v", todayCourses)

	// 更新缓存
	scheduleCache.Lock()
	scheduleCache.data[cacheKey] = todayCourses
	scheduleCache.Unlock()

	return todayCourses
}

func fetchFromU(room string) []UCourse {
	// 计算当前周数
	// 以2024年12月9日为第15周的周一为基准
	baseTime := time.Date(2024, 12, 9, 0, 0, 0, 0, time.Local)
	currentTime := time.Now()
	weekDiff := currentTime.Sub(baseTime).Hours() / (24 * 7)
	currentWeek := 15 + int(weekDiff)

	log.Printf("周数计算 - 基准时间: %v, 当前时间: %v, 计算得到的周数: %d",
		baseTime.Format("2006-01-02"),
		currentTime.Format("2006-01-02"),
		currentWeek)

	requestBody := map[string]interface{}{
		"type": "thisweek_courseschedule",
		"data": []map[string]interface{}{
			{
				"room_name": room,
				"qqdjz":     currentWeek,
			},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		log.Printf("Error marshaling request: %v", err)
		return nil
	}

	apiURL := getEnv("UESTC_API_URL", "")
	log.Printf("发送请求到 学校 API: %s", apiURL)
	log.Printf("请求体: %s", string(jsonBody))

	resp, err := http.Post(
		apiURL,
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		log.Printf("Error fetching data: %v", err)
		return nil
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return nil
	}
	log.Printf("UESTC API 响应: %s", string(respBody))

	// 重新创建一个新的 Reader 用于解码
	var response UResponse
	if err := json.NewDecoder(bytes.NewReader(respBody)).Decode(&response); err != nil {
		log.Printf("Error decoding response: %v", err)
		return nil
	}

	log.Printf("解析后的响应数据: %+v", response)
	return response.Data
}

func refreshCacheRoutine() {
	for {
		now := time.Now()
		hour := now.Hour()

		if hour >= 8 && hour <= 21 && hour%2 == 0 {
			// 清理缓存
			scheduleCache.Lock()
			scheduleCache.data = make(map[string][]UCourse)
			scheduleCache.Unlock()
		}

		// 等待到下一个整点
		nextCheck := time.Now().Add(time.Hour)
		nextCheck = time.Date(
			nextCheck.Year(),
			nextCheck.Month(),
			nextCheck.Day(),
			nextCheck.Hour(),
			0, 0, 0,
			nextCheck.Location(),
		)
		time.Sleep(time.Until(nextCheck))
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
