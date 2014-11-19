package main

import(
	"math"
	"fmt"
	"errors"
	"math/rand"
	"time"
)

type Result struct{
	Person Person
	Workload Workload
}

type Person struct{
	Vg float64 // Value/grade
	Gt float64 // Grade/time
	La float64 // Look ahead
	Vt float64 // Value/time
	P float64 // Rememberance
	B float64 // Beta
	D float64 // Delta

	Semester Semester // The schedule for the semester
	Assignments []Assignment // The assignment due
	WorkHours map[Assignment]int // The number of hours worked on assignments.
}

type Assignment struct{
	TotalGrade int // The total grade units this assignment is graded out of.
	DateDue int // The date due in numbers of day into the semester
}

func (a Assignment) String() string{
	return fmt.Sprintf("<A Total=%d Due=%d>",a.TotalGrade, a.DateDue)
}

type Semester struct{
	Days int // The number of days
	Day int // The current day
	Weights map[int]float64 // A mapping of days to the attitude weight.
	Allowed map[int]int // A maping of days to hours available to work.
}

type Work struct{
	Day int
	Hours int
	Assignment Assignment
}

func (w Work) String() string{
	return fmt.Sprintf("<W Day=%d Hours=%d Assignment=%v",w.Day, w.Hours, w.Assignment) 
}

func IntFactorial(i int)(int, error){
	if i > 20|| i < 1{return 0, errors.New("Factorial out of range")}
	if i == 1{
		return 1, nil
	}
	res, _ := IntFactorial(i-1)
	return i * res, nil
}

// Represents a possible ammount to work in the future.
type Workload struct{
	Days []Work
	Hours map[Assignment]int
}

func (w Workload) String() string {
	return fmt.Sprintf("<WL days=%v hours=%v>" ,w.Days, w.Hours)
}


// Generate workloads starting on the current day.
func (p *Person) Workloads() []Workload {

	workloads := make([]Workload,1)
	// Create base workload
	workloads[0] = *new(Workload)
	workloads[0].Days = make([]Work, 0)
	workloads[0].Hours = make(map[Assignment]int)
	for _,assignment := range(p.Assignments){
		workloads[0].Hours[assignment] = 0
	}
	// For each day until the look ahead day.
	for i := 0; float64(i) < p.La; i++{
		newloads := []Workload{}
		// For each workload
		for _,basewl := range(workloads){
			// For each assignment
			for _,assignment := range(p.Assignments){
				if assignment.DateDue >= (p.Semester.Day + i){
					// For every time that does not put you over the limit.
					for t:=1; basewl.Hours[assignment]+
						t+p.WorkHours[assignment] <=
						assignment.TotalGrade &&
						t <= p.Semester.Allowed[p.Semester.Day + i]; t++{

						wl := *new(Workload)
						day := *new(Work)
						day.Assignment = assignment
						day.Hours = t
						day.Day = i + p.Semester.Day
						wl.Days = append(basewl.Days,day)
						wl.Hours = make(map[Assignment]int)
						for k, v := range(basewl.Hours){
							wl.Hours[k]=v
						}

						wl.Hours[assignment] = basewl.Hours[assignment] + t
						newloads = append(newloads,wl)
					}
				}
			}
			wl := *new(Workload)
			day := *new(Work)
			day.Assignment = Assignment{}
			day.Hours = 0
			day.Day = i + p.Semester.Day
			wl.Days = append(basewl.Days,day)
			wl.Hours = make(map[Assignment]int)
			for k,v := range(basewl.Hours){
				wl.Hours[k]=v
			}
			newloads = append(newloads, wl)
		}
		
		workloads = newloads
	}
	return workloads
}

// The discount function for this individual
// If d is 0 then the function is 1, otherwise it is B * (D ^ d)
func (p *Person) Discount(d int) float64{
	if d == 0 {
		return 1
	}
	return p.B * math.Pow(p.D, float64(d))
}


// Returns the expected utility of the given work.
func (p Person) WorkUtility(work Work) float64{
	utility := (float64(work.Hours) * p.Vg *p.Gt *p.B * p.Discount(work.Assignment.DateDue - p.Semester.Day))
	cost := (float64(work.Hours) * p.Semester.Weights[work.Day] * p.Discount(work.Day - p.Semester.Day))
	return utility -cost
}

// Returns the expected utility of the given workload evaluated on the current day.
func (p Person) Utility(workload Workload) float64{
	utility := float64(0)
	for _, work := range(workload.Days){
		if work.Assignment.TotalGrade != 0{
			utility += p.WorkUtility(work)
		}
	}
	return utility
}

func (p Person) Simulate(results chan Result){
	var globalWorkload Workload 
	totalHours := 0
	for p.Semester.Day < p.Semester.Days {
		p.Semester.Day += 1
		maxUtility := float64(-100000)
		var choices []Workload
		choices = make([]Workload, 0)
		for _, workload := range(p.Workloads()){
			utility := p.Utility(workload)
			if utility == maxUtility && utility != 0{
				choices = append(choices, workload)
			}
			if utility > maxUtility{
				maxUtility = utility
				choices = make([]Workload, 0)
				choices = append(choices, workload)
			}
		}

		fmt.Printf("Choosing from {%d} choices\n", len(choices))
		var choice Workload
		if len(choices) > 1{
			fmt.Println(choices)
			choice = choices[rand.Intn(len(choices))]
		}else{
			choice = choices[0]
		}
		fmt.Printf("[%d]\t%f\t%v\n", p.Semester.Day,maxUtility, choice)
		if len(choice.Days) > 0 {
			globalWorkload.Days = append(globalWorkload.Days, choice.Days[0])
			p.WorkHours[choice.Days[0].Assignment] += choice.Days[0].Hours
			totalHours += choice.Days[0].Hours
		}
	}
	for assignment, hour := range(p.WorkHours){
		fmt.Printf("%d : %v\n",hour, assignment)
	}

	results <- Result{Person:p, Workload:globalWorkload}
}

func main(){
	fmt.Println("Simulating people... This is most likely not going to work.")

	person := Person{Vg:5, Gt:2, La: 7, Vt: 1, P:1, B:0.5, D:0.9, WorkHours:make(map[Assignment]int)}
	semester := Semester{Days: 30,Day:0,Weights:make(map[int]float64), Allowed:make(map[int]int)}
	person.Semester = semester
	for i := 0; i <= 30; i++{
		person.Semester.Weights[i] = float64(i)*0.065
		person.Semester.Allowed[i] = 2
		if i > 20 {
			//person.Semester.Weights[i] = float64(i - 20) * 0.75
			person.Semester.Allowed[i] = 1 
		}
	}
	rand.Seed(time.Now().Unix())
	person.Assignments = make([]Assignment, 0)
	person.Assignments = append(person.Assignments, Assignment{DateDue:20,TotalGrade: 10})
	person.Assignments = append(person.Assignments, Assignment{DateDue:30,TotalGrade: 5})
	person.Semester = semester
	
	results := make(chan Result)
	go person.Simulate(results)

	res := <- results
	fmt.Printf("Results: %v \n\t%v", res.Person, res.Workload)
}
