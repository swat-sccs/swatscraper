package main

import (
	"database/sql"
	"encoding/json"
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
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)


type termData struct {
	Count   int      `json:"totalCount"`
	Courses []course `json:"data"`
}

type course struct {
	ID              int               `json:"id" db:"courseId"`
	Ref             string            `json:"courseReferenceNumber" db:"courseReferenceNumber"`
	Number          string            `json:"courseNumber" db:"courseNumber"`
	Subject         string            `json:"subject" db:"subject"`
	Type            string            `json:"scheduleTypeDescription" db:"scheduleTypeDescription"`
	Title           string            `json:"courseTitle" db:"courseTitle"`
	DescriptionUrl  string            `json:"" db:"descriptionUrl"`
	Description     string            `json:"" db:"description"`
	Credits         float32            `json:"creditHours" db:"creditHours"`
	MaxEnrollment   int               `json:"maximumEnrollment" db:"maximumEnrollment"`
	Enrolled        int               `json:"enrollment" db:"enrollment"`
	Availability    int               `json:"seatsAvailable" db:"seatsAvailable"`
	Faculty         []faculty         `json:"faculty"`
	MeetingsFaculty []meetingsFaculty `json:"meetingsFaculty"`
	Attributes      []attribute       `json:"sectionAttributes"`
}

type faculty struct {
	ID    string `json:"bannerId" db:"bannerId"`
	Ref   string `json:"courseReferenceNumber" db:"courseReferenceNumber"`
	Name  string `json:"displayName" db:"displayName"`
	Email string `json:"emailAddress" db:"emailAddress"`
}

type meetingsFaculty struct {
	Section     string `json:"category" db:"category"`
	Ref         string `json:"courseReferenceNumber" db:"courseReferenceNumber"`
	MeetingTime meetingTime
}

type meetingTime struct {
	Begin         string  `json:"beginTime" db:"beginTime"`
	BuildingShort string  `json:"building" db:"building"`
	BuildingLong  string  `json:"buildingDescription" db:"buildingDescription"`
	Room          string  `json:"room" db:"room"`
	Section       string  `json:"category" db:"category"`
	Ref           string  `json:"courseReferenceNumber" db:"courseReferenceNumber"`
	EndDate       string  `json:"endDate" db:"endDate"`
	EndTime       string  `json:"endTime" db:"endTime"`
	StartDate     string  `json:"startDate" db:"startDate"`
	Hours         float32  `json:"hoursWeek" db:"hoursWeek"`
	TypeShort     string  `json:"meetingType" db:"meetingType"`
	TypeLong      string  `json:"meetingTypeDescription" db:"meetingTypeDescription"`
	Monday        bool    `json:"monday" db:"monday"`
	Tuesday       bool    `json:"tuesday" db:"tuesday"`
	Wednesday     bool    `json:"wednesday" db:"wednesday"`
	Thursday      bool    `json:"thursday" db:"thursday"`
	Friday        bool    `json:"friday" db:"friday"`
	Saturday      bool    `json:"saturday" db:"saturday"`
	Sunday        bool    `json:"sunday" db:"sunday"`
}

type attribute struct {
	CodeShort string `json:"code" db:"code`
	CodeLong  string `json:"description" db:"description`
	Ref       string `json:"courseReferenceNumber" db:"courseReferenceNumber`
}

type attribute_list struct{
	attributes []attribute  
}


func load_envs (){
	// load .env file
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	
	var	host     = os.Getenv("HOST")
	var	port     = 5432
	var	user     = os.Getenv("SQL_USER")
	var	password = os.Getenv("PASS")
	var	dbname   = os.Getenv("DBNAME")


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

func send_to_db(data termData){

	var MeetingTimeID int;
	var MeetingsFacultyID int;
	var facultyID int;
	var courseID int;
	var sectionAttributeID int;

	// load .env file
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	
	var	host     = os.Getenv("HOST")
	var	port     = 5432
	var	user     = os.Getenv("SQL_USER")
	var	password = os.Getenv("PASS")
	var	dbname   = os.Getenv("DBNAME")


	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+"password=%s dbname=%s sslmode=disable", host, port, user, password,dbname)
	db, err := sql.Open("postgres", psqlInfo)

	//db, err := sqlx.Connect("postgres", psqlInfo)
	
	if err != nil {
		panic(err)
	}
	
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	fmt.Println("Successfully connected to DB!")

	for k := range data.Courses {
		//FACULTY
		if(len(data.Courses[k].Faculty) > 0){
			facultyID = 0
			sqlStatement1 := `INSERT INTO "Faculty" ("bannerId", 
													"courseReferenceNumber", 
													"displayName", 
													"emailAddress")

													VALUES ($1,
															$2,
															$3,
															$4
															) RETURNING id;`
								
			err := db.QueryRow(sqlStatement1,
				data.Courses[k].Faculty[0].ID,
				data.Courses[k].Faculty[0].Ref ,
				data.Courses[k].Faculty[0].Name ,
				data.Courses[k].Faculty[0].Email).Scan(&facultyID)
			if err != nil {
				panic(err)
			}	
			
			//fmt.Print("THINGS: ", facultyId)
		}

		//MEETING TIME
		if(len(data.Courses[k].MeetingsFaculty) > 0){
			MeetingTimeID = 0
			sqlStatement2 := `INSERT INTO "MeetingTime" ("beginTime", 
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
														"sunday")


											VALUES ($1,
													$2,
													$3,
													$4,
													$5,
													$6,
													$7,
													$8,
													$9,
													$10,
													$11,
													$12,
													$13,
													$14,
													$15,
													$16,
													$17,
													$18,
													$19
													) RETURNING id;`
								
			err := db.QueryRow(sqlStatement2,
				data.Courses[k].MeetingsFaculty[0].MeetingTime.Begin,
				data.Courses[k].MeetingsFaculty[0].MeetingTime.BuildingShort ,
				data.Courses[k].MeetingsFaculty[0].MeetingTime.BuildingLong ,
				data.Courses[k].MeetingsFaculty[0].MeetingTime.Room,
				data.Courses[k].MeetingsFaculty[0].MeetingTime.Section,
				data.Courses[k].MeetingsFaculty[0].MeetingTime.Ref,
				data.Courses[k].MeetingsFaculty[0].MeetingTime.EndDate,
				data.Courses[k].MeetingsFaculty[0].MeetingTime.EndTime,
				data.Courses[k].MeetingsFaculty[0].MeetingTime.StartDate,
				data.Courses[k].MeetingsFaculty[0].MeetingTime.Hours,
				data.Courses[k].MeetingsFaculty[0].MeetingTime.TypeShort,
				data.Courses[k].MeetingsFaculty[0].MeetingTime.TypeLong,
				data.Courses[k].MeetingsFaculty[0].MeetingTime.Monday,
				data.Courses[k].MeetingsFaculty[0].MeetingTime.Tuesday,
				data.Courses[k].MeetingsFaculty[0].MeetingTime.Wednesday,
				data.Courses[k].MeetingsFaculty[0].MeetingTime.Thursday,
				data.Courses[k].MeetingsFaculty[0].MeetingTime.Friday,
				data.Courses[k].MeetingsFaculty[0].MeetingTime.Saturday,
				data.Courses[k].MeetingsFaculty[0].MeetingTime.Sunday,
				
				).Scan(&MeetingTimeID)
			if err != nil {
				panic(err)
			}	
			
		}

		//Meetings Faculty
		if(len(data.Courses[k].MeetingsFaculty) > 0){
			MeetingsFacultyID = 0
			sqlStatement2 := `INSERT INTO "MeetingsFaculty" ("category", 
															 "courseReferenceNumber",
															 "meetingTimeID")


													VALUES ($1,
															$2,
															$3) RETURNING id;`
								
			err := db.QueryRow(sqlStatement2,
				data.Courses[k].MeetingsFaculty[0].Section,
				data.Courses[k].MeetingsFaculty[0].Ref ,
				MeetingTimeID,
				).Scan(&MeetingsFacultyID)
			if err != nil {
				panic(err)
			}	
		}


		//Full Course insert
		courseID = 0
		sqlStatement2 := `INSERT INTO "Course"("courseId",                
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
													"facultyMeetId"                                         
													)


										VALUES ($1,
												$2,
												$3,
												$4,
												$5,
												$6,
												$7,
												$8,
												$9,
												$10,
												$11,
												$12,
												$13,
												$14
												) RETURNING id;`
							
		err := db.QueryRow(sqlStatement2,
			data.Courses[k].ID,
			data.Courses[k].Ref ,
			data.Courses[k].Number ,
			data.Courses[k].Subject,
			data.Courses[k].Type,
			data.Courses[k].Title,
			data.Courses[k].DescriptionUrl,
			data.Courses[k].Description,
			data.Courses[k].Credits,
			data.Courses[k].MaxEnrollment,
			data.Courses[k].Enrolled,
			data.Courses[k].Availability,
			facultyID,
			MeetingsFacultyID,
			).Scan(&courseID)
		if err != nil {
			panic(err)
		}	


		//Attributes
		if(len(data.Courses[k].Attributes) > 0){
			sectionAttributeID = 0
			sqlStatement2 := `INSERT INTO "sectionAttribute" ("code", 
															 "description",
															 "courseReferenceNumber",
															 "courseId")


													VALUES ($1,
															$2,
															$3,
															$4 ) RETURNING id;`
								
			err := db.QueryRow(sqlStatement2,
				data.Courses[k].Attributes[0].CodeShort,
				data.Courses[k].Attributes[0].CodeLong ,
				data.Courses[k].Attributes[0].Ref ,
				courseID,
				).Scan(&sectionAttributeID)
			if err != nil {
				panic(err)
			}	
		}
			


		time.Sleep(100 * time.Millisecond) 
	}
}


func timer(name string) func() {
	start := time.Now()
	return func() {
		fmt.Printf("%s took %v\n", name, time.Since(start))
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
	var semester, year string
	var wg sync.WaitGroup

	//load_envs()

	fmt.Print("Enter your semester (i.e. fall): ")
	fmt.Scan(&semester)

	fmt.Print("Enter your year (i.e. 2024): ")
	fmt.Scan(&year)

	term := setTerm(semester, year)

	defer timer("main")()

	jar, _ := cookiejar.New(nil)

	client := http.Client{
		Jar: jar,
	}

	client.Get("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/registration")
	client.Get("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/term/termSelection?mode=search")
	client.Get("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/classSearch/getTerms?searchTerm=&offset=1&max=10&_=1717271345154")
	client.Get("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/term/search?mode=search&term=202404&studyPath=&studyPathText=&startDatepicker=&endDatepicker=&uniqueSessionId=l47z91717271338036")

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
			fmt.Println("Finished processing:", data.Count, "courses")
			break
		}
	}

	getCourseDescriptionUrls(term, data)

	for i := range data.Courses {
		wg.Add(1)
		go requestCourseDescription(i, data, client, &wg)
	}

	wg.Wait()

	// output, err := json.MarshalIndent(data, "", "\t")

	// if err != nil {
	// 	fmt.Println(err)
	// }

	fmt.Println("Reformatting JSON")

	courseMap := make(map[int]course)

	for k := range data.Courses {
		courseMap[data.Courses[k].ID] = data.Courses[k]
	}

	output, err := json.MarshalIndent(courseMap, "", "\t")

	if err != nil {
		fmt.Println(err)
	}

	send_to_db(data)
	

	err = os.WriteFile("courses.json", output, 0644)

	if err != nil {
		fmt.Println(err)
	}

}
