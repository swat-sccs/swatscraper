package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gtuk/discordwebhook"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
)

type termData struct {
	Count   int      `json:"totalCount"`
	Courses []course `json:"data"`
}

type course struct {
	ID               int               `json:"id" db:"courseId"`
	Ref              string            `json:"courseReferenceNumber" db:"courseReferenceNumber"`
	Number           string            `json:"courseNumber" db:"courseNumber"`
	Subject          string            `json:"subject" db:"subject"`
	Type             string            `json:"scheduleTypeDescription" db:"scheduleTypeDescription"`
	Title            string            `json:"courseTitle" db:"courseTitle"`
	DescriptionUrl   string            `json:"" db:"descriptionUrl"`
	Description      string            `json:"" db:"description"`
	Credits          float32           `json:"creditHours" db:"creditHours"`
	MaxEnrollment    int               `json:"maximumEnrollment" db:"maximumEnrollment"`
	Enrolled         int               `json:"enrollment" db:"enrollment"`
	Availability     int               `json:"seatsAvailable" db:"seatsAvailable"`
	Faculty          []faculty         `json:"faculty"`
	MeetingsFaculty  []meetingsFaculty `json:"meetingsFaculty"`
	Attributes       []attribute       `json:"sectionAttributes"`
	Year             string            `db:"year"`
	LinkedSectionUrl string            `json:""`
	IsSectionLinked  bool              `json:"isSectionLinked"`
	LinkedSections   string            `db:"linkedSections"`
	FacultyID        int               `db:"facultyId"`
	FacultyMeetID    int               `db:"facultyMeetId"`
}

type faculty struct {
	ID    string `json:"bannerId" db:"bannerId"`
	Ref   string `json:"courseReferenceNumber" db:"courseReferenceNumber"`
	Name  string `json:"displayName" db:"displayName"`
	Email string `json:"emailAddress" db:"emailAddress"`
	Year  string `db:"year"`
}

type meetingsFaculty struct {
	Section       string `json:"category" db:"category"`
	Ref           string `json:"courseReferenceNumber" db:"courseReferenceNumber"`
	Year          string `db:"year"`
	MeetingTimeID int    `db:"meetingTimeID"`
	MeetingTime   meetingTime
}

type meetingTime struct {
	Begin         string    `json:"beginTime" db:"beginTime"`
	BEGINTIME     time.Time `db:"BEGINTIME"`
	ENDTIME       time.Time `db:"ENDTIME"`
	BuildingShort string    `json:"building" db:"building"`
	BuildingLong  string    `json:"buildingDescription" db:"buildingDescription"`
	Room          string    `json:"room" db:"room"`
	Section       string    `json:"category" db:"category"`
	Ref           string    `json:"courseReferenceNumber" db:"courseReferenceNumber"`
	EndDate       string    `json:"endDate" db:"endDate"`
	EndTime       string    `json:"endTime" db:"endTime"`
	StartDate     string    `json:"startDate" db:"startDate"`
	Hours         float32   `json:"hoursWeek" db:"hoursWeek"`
	TypeShort     string    `json:"meetingType" db:"meetingType"`
	TypeLong      string    `json:"meetingTypeDescription" db:"meetingTypeDescription"`
	Monday        bool      `json:"monday" db:"monday"`
	Tuesday       bool      `json:"tuesday" db:"tuesday"`
	Wednesday     bool      `json:"wednesday" db:"wednesday"`
	Thursday      bool      `json:"thursday" db:"thursday"`
	Friday        bool      `json:"friday" db:"friday"`
	Saturday      bool      `json:"saturday" db:"saturday"`
	Sunday        bool      `json:"sunday" db:"sunday"`
	Year          string    `db:"year"`
}

type linkedSectionsList struct {
	LinkedData [][]struct {
		CourseReferenceNumber string `json:"courseReferenceNumber"`
	} `json:"linkedData"`
}

type attribute struct {
	CodeShort string `json:"code" db:"code"`
	CodeLong  string `json:"description" db:"description"`
	Ref       string `json:"courseReferenceNumber" db:"courseReferenceNumber"`
	Year      string `db:"year"`
	CourseID  int    `db:"courseId"`
}

type attribute_list struct {
	attributes []attribute
}

var semester, year string
var classCount int

func load_envs() {
	// load .env file
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	var host = os.Getenv("HOST")
	var port = 5432
	var user = os.Getenv("SQL_USER")
	var password = os.Getenv("PASS")
	var dbname = os.Getenv("DBNAME")

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	fmt.Println("Successfully connected to DB!")
}

func send_to_db(data termData, semester string, year string) {
	var MeetingTimeID int
	var MeetingsFacultyID int
	var facultyID int
	var courseID int
	var sectionAttributeID int
	var yearString string

	if strings.ToLower(semester) == "fall" {
		yearString = "F" + year
	} else {
		yearString = "S" + year
	}
	fmt.Println(yearString)

	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	var host = os.Getenv("HOST")
	var port = 5432
	var user = os.Getenv("SQL_USER")
	var password = os.Getenv("PASS")
	var dbname = os.Getenv("DBNAME")

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+"password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

	db := sqlx.MustConnect("postgres", psqlInfo)
	db.SetMaxOpenConns(80)

	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	fmt.Println("Successfully connected to DB!")

	for k := range data.Courses {
		if len(data.Courses[k].Faculty) > 0 {
			query := `INSERT INTO "Faculty"("bannerId", 

													"displayName", 
													"emailAddress",
													"year")

													VALUES (:bannerId,

															:displayName,
															:emailAddress,
															:year) 
							ON CONFLICT ("bannerId")
							DO UPDATE SET 
								"bannerId" = :bannerId, "displayName"= :displayName, "emailAddress"= :emailAddress RETURNING id;`

			var data1 faculty = data.Courses[k].Faculty[0]
			data1.Ref = yearString + data1.Ref
			data1.Year = yearString

			rows, err := db.NamedQuery(query, data1)
			if err != nil {
				log.Fatalln(err)
			}
			if rows.Next() {
				rows.Scan(&facultyID)
			}
			rows.Close()
		}

		//Meeting Time
		if len(data.Courses[k].MeetingsFaculty) > 0 {
			MeetingTimeID = 0
			query := `INSERT INTO "MeetingTime" ("beginTime", 
												"building", 
												"buildingDescription", 
												"room",
												"category",               
												"courseReferenceNumber",  
												"endDate",                
												"endTime",                
												"startDate",              
												"hoursWeek",           
												"meetingType",            
												"meetingTypeDescription", 
												"monday",                 
												"tuesday",                
												"wednesday",              
												"thursday",               
												"friday",                 
												"saturday",               
												"sunday",
												"year")


											VALUES (:beginTime, 
													:building, 
													:buildingDescription, 
													:room,
													:category,               
													:courseReferenceNumber,  
													:endDate,                
													:endTime,                
													:startDate,              
													:hoursWeek,           
													:meetingType,            
													:meetingTypeDescription, 
													:monday,                 
													:tuesday,                
													:wednesday,              
													:thursday,               
													:friday,                 
													:saturday,               
													:sunday,
													:year)
							ON CONFLICT ("courseReferenceNumber")
							DO UPDATE SET 
								"beginTime" = :beginTime, 
								"building" = :building, 
								"buildingDescription" = :buildingDescription, 
								"room" = :room,
								"category" = :category,               
								"courseReferenceNumber" = :courseReferenceNumber,  
								"endDate" = :endDate,                
								"endTime" = :endTime,                
								"startDate" = :startDate,              
								"hoursWeek" = :hoursWeek,           
								"meetingType" = :meetingType,            
								"meetingTypeDescription" = :meetingTypeDescription, 
								"monday" = :monday,                 
								"tuesday" = :tuesday,                
								"wednesday" = :wednesday,              
								"thursday" = :thursday,               
								"friday"= :friday,                 
								"saturday"= :saturday,               
								"sunday" = :sunday ,
								"year"= :year
								RETURNING id;`

			var data1 meetingTime = data.Courses[k].MeetingsFaculty[0].MeetingTime

			if data1.Begin != "" {
				var beginTime = data1.Begin
				hour, _ := strconv.Atoi(beginTime[0:2])
				min, _ := strconv.Atoi(beginTime[2:4])
				theTime := time.Date(2005, 05, 15, hour, min, 00, 00, time.UTC)
				leTime, _ := time.Parse(time.RFC3339Nano, theTime.Format(time.RFC3339))

				data1.BEGINTIME = leTime
			}
			if data1.EndTime != "" {
				var EndTime = data1.EndTime
				hour, _ := strconv.Atoi(EndTime[0:2])
				min, _ := strconv.Atoi(EndTime[2:4])
				theTime := time.Date(2005, 05, 15, hour, min, 00, 00, time.UTC)
				leTime, _ := time.Parse(time.RFC3339, theTime.Format(time.RFC3339))

				data1.ENDTIME = leTime
			}

			data1.Ref = yearString + data1.Ref
			data1.Year = yearString

			rows, err := db.NamedQuery(query, data1)
			if err != nil {
				log.Fatalln(err)
			}
			if rows.Next() {
				rows.Scan(&MeetingTimeID)
			}
			rows.Close()
		}

		//Meeting Faculty
		if len(data.Courses[k].MeetingsFaculty) > 0 {
			MeetingsFacultyID = 0
			query := `INSERT INTO "MeetingsFaculty" ("category", 
													"courseReferenceNumber",
													"meetingTimeID",
													"year")


													VALUES (:category,
															:courseReferenceNumber,
															:meetingTimeID,
															:year)
							ON CONFLICT ("courseReferenceNumber")
							DO UPDATE SET 
								"category" = :category, "meetingTimeID"= :meetingTimeID , "year" = :year RETURNING id;`

			var data1 meetingsFaculty = data.Courses[k].MeetingsFaculty[0]
			data1.Ref = yearString + data1.Ref
			data1.Year = yearString
			data1.MeetingTimeID = MeetingTimeID

			rows, err := db.NamedQuery(query, data1)
			if err != nil {
				log.Fatalln(err)
			}
			if rows.Next() {
				rows.Scan(&MeetingsFacultyID)
			}
			rows.Close()
		}

		courseID = 0
		query := `INSERT INTO "Course"("courseId",                
										"courseReferenceNumber",   
										"courseNumber",            
										"subject",                 
										"scheduleTypeDescription", 
										"courseTitle",             
										"descriptionUrl",          
										"description",             
										"creditHours",             
										"maximumEnrollment",       
										"enrollment",              
										"seatsAvailable",          
										"facultyId",               
										"facultyMeetId",
										"year",
										"linkedSections"                          
										)


										VALUES (:courseId,                
												:courseReferenceNumber,   
												:courseNumber,            
												:subject,                 
												:scheduleTypeDescription, 
												:courseTitle,             
												:descriptionUrl,          
												:description,             
												:creditHours,             
												:maximumEnrollment,       
												:enrollment,              
												:seatsAvailable,          
												:facultyId,               
												:facultyMeetId,
												:year,
												:linkedSections
												)
							ON CONFLICT ("courseReferenceNumber")
							DO UPDATE SET 
										"courseId" = :courseId,                
										"courseNumber" = :courseNumber,            
										"subject" = :subject,                 
										"scheduleTypeDescription" = :scheduleTypeDescription, 
										"courseTitle" = :courseTitle,             
										"descriptionUrl" = :descriptionUrl,          
										"description" = :description,             
										"creditHours" = :creditHours,             
										"maximumEnrollment" = :maximumEnrollment,       
										"enrollment" = :enrollment,              
										"seatsAvailable" = :seatsAvailable,          
										"facultyId" = :facultyId,               
										"facultyMeetId" = :facultyMeetId,
										"linkedSections" = :linkedSections,
										"year" = :year RETURNING id;`

		var data1 course = data.Courses[k]
		data1.Ref = yearString + data1.Ref
		data1.FacultyID = facultyID
		data1.FacultyMeetID = MeetingsFacultyID
		data1.Year = yearString

		rows, err := db.NamedQuery(query, data1)
		if err != nil {
			log.Fatalln(err)
		}
		if rows.Next() {
			rows.Scan(&courseID)
		}
		rows.Close()

		//Attributes
		if len(data.Courses[k].Attributes) > 0 {
			sectionAttributeID = 0
			query := `INSERT INTO "sectionAttribute" ("code", 
													"description",
													"courseReferenceNumber",
													"courseId",
													"year")

													VALUES (:code,
															:description,
															:courseReferenceNumber,
															:courseId ,
															:year) 
													ON CONFLICT ("courseReferenceNumber")
													DO UPDATE SET 
																"code" = :code,
																"description" = :description,                
																"courseId" = :courseId     ,
																"year" = :year      
																RETURNING id;`

			var data1 = data.Courses[k].Attributes
			for z := range data1 {

				// perform an operation
				data1[z].Ref = yearString + data1[z].Ref + data1[z].CodeShort
				data1[z].CourseID = courseID
				data1[z].Year = yearString

				rows, err = db.NamedQuery(query, data1[z])
				if err != nil {
					log.Fatalln(err)
				}
				if rows.Next() {
					rows.Scan(&sectionAttributeID)
				}
				rows.Close()
			}

		}
	}
}

func timer(name string) func() {
	start := time.Now()
	//var username = "Scraper Bot"
	var color = "14177041"
	var username = "Swat Scraper"
	var iconURL = "https://images.sidearmdev.com/convert?url=https%3a%2f%2fdxbhsrqyrr690.cloudfront.net%2fsidearm.nextgen.sites%2fswarthmoreathletics.com%2fimages%2f2021%2f5%2f21%2fsc_logo_g2_rgb.png&type=webp"
	var truer = true;
	var countTitle = "Course count"
	var timeName = "Scrape Time"
	

	return func() {
		fmt.Printf("%s took %v\n", name, time.Since(start))
		//var content  = "Scrape of (" + semester + ", " + year + ") took " + time.Since(start).String()
		var countContent = strconv.Itoa(classCount)
		var title = semester + ", " + year
		var timeContent = time.Since(start).String()

		var opMode = "OP Mode"
		var opModeContent = os.Getenv("OPMODE")

		field := discordwebhook.Field{
			Name:&timeName,
			Value:&timeContent,
			Inline: &truer ,
		}
		field2 := discordwebhook.Field{
			Name:&opMode,
			Value:&opModeContent,
			Inline: &truer ,
		}

		field3 := discordwebhook.Field{
			Name:&countTitle,
			Value:&countContent,
			Inline: &truer ,
		}

		author := discordwebhook.Author{
			Name:&username, 
			IconUrl: &iconURL,
		}

	

		embed := discordwebhook.Embed{
			Title: &title,
			Author: &author,
			Color: &color,
			Fields: &[]discordwebhook.Field{field2,field3,field},
		}
		
		message := discordwebhook.Message{
			Username: &username,
			Embeds:   &[]discordwebhook.Embed{embed},

		}
			
		err := discordwebhook.SendMessage(os.Getenv("WEBHOOK"), message)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func setTerm(semester string, year string) string {
	var term strings.Builder

	term.WriteString(year)

	if semester == "fall" {
		term.WriteString("04")
	} else {
		term.WriteString("02")
	}

	return term.String()
}

func requestCourses(term string, offset string, max string, client http.Client) (*termData, error) {
	// Note: Endpoint is limited to 500 courses per request, we'll use some sort of pagination
	// Will probably not exceed 1000 courses so, for now, 2 requests will be enough

	var swarthmoreUrl strings.Builder

	swarthmoreUrl.WriteString("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/searchResults/searchResults?txt_term=")
	swarthmoreUrl.WriteString(term)
	swarthmoreUrl.WriteString("&startDatepicker=&endDatepicker=&uniqueSessionId=cwtoq1717225731537&pageOffset=")
	swarthmoreUrl.WriteString(offset)
	swarthmoreUrl.WriteString("&pageMaxSize=")
	swarthmoreUrl.WriteString(max)
	swarthmoreUrl.WriteString("&sortColumn=subjectDescription&sortDirection=asc")

	resp, err := client.Get(swarthmoreUrl.String())

	if err != nil {
		return nil, fmt.Errorf("failed to fulfill GET request: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	data := new(termData)

	if err := json.Unmarshal(body, data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return data, nil
}

func getCourseDescriptionUrls(term string, data termData) {
	for i := range data.Courses {
		var formattedUrl strings.Builder

		formattedUrl.WriteString("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/searchResults/getCourseDescription?term=")
		formattedUrl.WriteString(term)
		formattedUrl.WriteString("&courseReferenceNumber=")
		formattedUrl.WriteString(data.Courses[i].Ref)

		url := formattedUrl.String()

		data.Courses[i].DescriptionUrl = url
	}
}

func getCourseLinkedSectionsUrls(term string, data termData) {
	for i := range data.Courses {
		var formattedUrl strings.Builder

		formattedUrl.WriteString("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/searchResults/fetchLinkedSections?term=")
		formattedUrl.WriteString(term)
		formattedUrl.WriteString("&courseReferenceNumber=")
		formattedUrl.WriteString(data.Courses[i].Ref)

		url := formattedUrl.String()

		data.Courses[i].LinkedSectionUrl = url
	}
}

func requestCourseLinkedSections(index int, data termData, client http.Client, wg *sync.WaitGroup) {
	defer wg.Done()

	if data.Courses[index].IsSectionLinked {
		resp, err := client.Get(data.Courses[index].LinkedSectionUrl)

		if err != nil {
			log.Println(err)
		}

		if err != nil {
			fmt.Println("Error fetching linked sections:", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Println("Error: Received non-200 HTTP status:", resp.StatusCode)
			return
		}

		body, err := io.ReadAll(resp.Body)

		if err != nil {
			fmt.Println("Error reading response body:", err)
			return
		}

		var linkedSections linkedSectionsList

		if err := json.Unmarshal(body, &linkedSections); err != nil {
			fmt.Printf("Failed to unmarshal JSON: %v\n", err)
			return
		}

		var courseReferenceNumbers []string

		for _, elem := range linkedSections.LinkedData {
			courseReferenceNumbers = append(courseReferenceNumbers, elem[0].CourseReferenceNumber)
		}

		data.Courses[index].LinkedSections = strings.Join(courseReferenceNumbers, ",")
	}
}

func requestCourseDescription(index int, data termData, client http.Client, wg *sync.WaitGroup) {
	defer wg.Done()

	resp, err := client.Get(data.Courses[index].DescriptionUrl)

	if err != nil {
		fmt.Println(err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		fmt.Println(err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))

	if err != nil {
		fmt.Println(err)
	}

	section := doc.Find(`section[aria-labelledby="courseDescription"]`)
	_, formattedString, ok := strings.Cut(section.Text(), "Section information text:")

	if !ok {
		data.Courses[index].Description = "No course description provided. Contact Professor."
	} else {
		data.Courses[index].Description = strings.TrimSpace(formattedString)
	}
}

func main() {
	
	var wg sync.WaitGroup

	flag.StringVar(&semester, "semester", "fall", "The semster to scrape")
	flag.StringVar(&year, "year", "2024", "The year to scrape")
	flag.Parse()

	term := setTerm(semester, year)

	defer timer("main")()

	jar, _ := cookiejar.New(nil)

	client := http.Client{
		Jar: jar,
	}

	var formattedUrl strings.Builder

	formattedUrl.WriteString("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/term/search?mode=search&term=")
	formattedUrl.WriteString(term)
	formattedUrl.WriteString("&studyPath=&studyPathText=&startDatepicker=&endDatepicker=&uniqueSessionId=l47z91717271338036")

	client.Get("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/registration")
	client.Get("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/term/termSelection?mode=search")
	client.Get("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/classSearch/getTerms?searchTerm=&offset=1&max=10&_=1717271345154")
	client.Get(formattedUrl.String())

	fmt.Println("Hydrated client")

	processedCourses := 0

	var data termData

	fmt.Println("Requesting courses")

	for {
		if processedCourses == 0 {
			courses, err := requestCourses(term, "0", "500", client)

			if err != nil {
				fmt.Println(err)
				fmt.Println("retrying in 7 seconds")
				time.Sleep(7000 * time.Millisecond)
			}

			data.Courses = append(data.Courses, courses.Courses...)
			data.Count = courses.Count

		} else {
			courses, err := requestCourses(term, strconv.Itoa(processedCourses), "500", client)

			if err != nil {
				fmt.Println("retrying in 7 seconds")
				time.Sleep(7000 * time.Millisecond)
			}

			data.Courses = append(data.Courses, courses.Courses...)
		}

		processedCourses += 500

		if processedCourses >= data.Count {
			classCount = data.Count;
			fmt.Println("Finished processing:", data.Count, "courses")
			break
		}
	}

	getCourseDescriptionUrls(term, data)
	getCourseLinkedSectionsUrls(term, data)

	for i := range data.Courses {
		wg.Add(2)

		go requestCourseDescription(i, data, client, &wg)
		go requestCourseLinkedSections(i, data, client, &wg)
	}

	wg.Wait()

	fmt.Println("Reformatting JSON")

	courseMap := make(map[int]course)

	for k := range data.Courses {
		courseMap[data.Courses[k].ID] = data.Courses[k]
	}

	output, err := json.MarshalIndent(courseMap, "", "\t")

	if err != nil {
		fmt.Println(err)
	}

	err = os.WriteFile("courses.json", output, 0644)

	send_to_db(data, semester, year)

	if err != nil {
		fmt.Println(err)
	}
}
