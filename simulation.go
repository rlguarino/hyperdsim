package main

import(
	"math"
	"fmt"
	"errors"
	"math/rand"
	"time"
	"sync"
	"code.google.com/p/plotinum/plot"
	"code.google.com/p/plotinum/plotter"
	"code.google.com/p/plotinum/plotutil"
)

type Result struct{
	Person Person
	Workload Workload
}

func (r Result) String()string{
	return fmt.Sprintf("<Result %v %s>", r.Person, r.Workload)
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

func (p Person) String() string{
	return fmt.Sprintf("<Person Vg:%f Gt:%f La:%f Vt:%f P:%f B:%f D:%f",
		p.Vg, p.Gt, p.La, p.Vt, p.P, p.B, p.D)
}

type Assignment struct{
	TotalGrade int // The total grade units this assignment is graded out of.
	DateDue int // The date due in numbers of day into the semester
	Name string
}

func (a Assignment) String() string{
	return fmt.Sprintf("<A Total=%d Due=%d Name:%s>",a.TotalGrade, a.DateDue, a.Name)
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
	globalWorkload.Hours = make(map[Assignment]int)

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

		var choice Workload
		if len(choices) > 1{
			choice = choices[rand.Intn(len(choices))]
		}else{
			choice = choices[0]
		}
		if len(choice.Days) > 0 {
			globalWorkload.Days = append(globalWorkload.Days, choice.Days[0])
			p.WorkHours[choice.Days[0].Assignment] += choice.Days[0].Hours
			globalWorkload.Hours[choice.Days[0].Assignment] += choice.Days[0].Hours
		}
	}

	results <- Result{Person:p, Workload:globalWorkload}
}

func main(){
	fmt.Println("Simulating people... This is most likely not going to work.")

	people := make([]Person, 0)
	rand.Seed(time.Now().Unix())
	
	assignments := make([]Assignment, 0)
	assignments = append(assignments, Assignment{DateDue:30,TotalGrade: 20, Name:"Assignment 1"})
	assignments = append(assignments, Assignment{DateDue:30,TotalGrade: 20, Name:"Assignment 2"})
	numPeople := 100
	numDays := 30
	for p:=0; p< numPeople; p++{
		// Create person
		Vg := rand.NormFloat64() * 1 + 10
		Gt := rand.NormFloat64() * 0.25 + 2
		B := rand.NormFloat64() * 0.2 + 0.5
		D := rand.NormFloat64() * 0.1 + 0.8
		if D > 1.0{
			D = 1
		}
		person := Person{Vg:Vg, Gt:Gt, La: 7, Vt: 1, P:1, B:B, D:D, WorkHours:make(map[Assignment]int)}
		semester := Semester{Days: 30,Day:0,Weights:make(map[int]float64), Allowed:make(map[int]int)}
		person.Semester = semester
		for i := 0; i <= 30; i++{
			person.Semester.Weights[i] = float64(i)*0.025
			person.Semester.Allowed[i] = 3
			if i > 25 {
				person.Semester.Allowed[i] = 2
				person.Semester.Weights[i] += float64(i)*0.025
			}
		}

		person.Assignments = assignments

		people = append(people, person)
	}

	results := make(chan Result)
	inChan := make(chan Person)

	numRoutines := 10
	var wg sync.WaitGroup
	for i := 0; i < numRoutines; i++{
		wg.Add(1)
		go func(people chan Person, results chan Result){
			for person := range(people){
				person.Simulate(results)
			}
			wg.Done()
		}(inChan, results)
	}

	//Wait for the channel simulations to finish then close the results chan
	go func(){
		wg.Wait()
		close(results)
	}()

	// Keep placing people in the channel
	go func(people []Person, inChan chan Person){
		for _,person := range people{
			inChan <- person
		}
		close(inChan)
	}(people, inChan)
	
	frequency := make(map[Assignment]plotter.XYs, len(assignments))
	grades := make(map[Assignment]int)
	gradeDistribution := make(plotter.XYs,101)
	
	for i := 0;i<= 100;i++{
		gradeDistribution[i].X = float64(i)
		gradeDistribution[i].Y = 0
	}

	for _, Assignment := range assignments{
		frequency[Assignment] = make(plotter.XYs,numDays+1)
		for i := 0;i <= numDays; i++{
			frequency[Assignment][i].X = float64(i)
			frequency[Assignment][i].Y = 0
		}
	}
	z :=0
	for res := range results{
		res := res
		fmt.Printf("Results: %v \n\t%v\n", res.Person, res.Workload)
		for n,day := range res.Workload.Days{
			if day.Assignment.TotalGrade != 0{
				frequency[day.Assignment][n].Y += float64(day.Hours)
			}
		}
		for Assignment, Grade := range res.Workload.Hours{
			if Assignment.Name != ""{ 
				grade := int((float64(Grade)/(float64(Assignment.TotalGrade))*float64(100)))
				gradeDistribution[grade].Y = gradeDistribution[grade].Y + float64(1)
				grades[Assignment] = grades[Assignment] + Grade
			}
		}
		z+=1
	}
	wt, err := plot.New()
	if err != nil {
			panic(err)
	}
	wt.Title.Text = "Distribution of work hours vs time."
	wt.Y.Label.Text = "Work Hours"
	wt.X.Label.Text = "Time"
	wt.Add(plotter.NewGrid())

	for assignment,hour := range grades{
		fmt.Printf("Average %s: %f\n", assignment, float64(hour)/float64(numPeople))
	}

	i := 0
	for assignment, xys := range frequency{
		lpline,lppoints, err := plotter.NewLinePoints(xys)
		if err != nil{
			panic(err)
		}
		lpline.Color = plotutil.Color(i)
		lpline.Dashes = plotutil.Dashes(i)
		lppoints.Color = plotutil.Color(i)
		lppoints.Shape = plotutil.Shape(i)
		i++
		wt.Add(lpline)
		wt.Add(lppoints)
		wt.Legend.Add(assignment.Name, lpline, lppoints)
	}
	wt.Legend.Top = true
	wt.X.Min=0
	wt.X.Max=float64(numDays)
	if err := wt.Save(5, 3, "workDistribution.png"); err != nil {
		panic(err)
	}

	gt, err := plot.New()
	if err != nil {
			panic(err)
	}
	gt.Title.Text = "Distribution of grades"
	gt.Y.Label.Text = "Frequency"
	gt.X.Label.Text = "Grades"
	gt.Add(plotter.NewGrid())

	h, err := plotter.NewHistogram(gradeDistribution, 10)
	if err != nil{
		panic(err)
	}
	h.FillColor = plotutil.Color(1)
	gt.Add(h)

	gt.Legend.Top = true
	gt.Y.Min=0
	gt.Y.Tick.Marker = func(min, max float64) []plot.Tick {
		const suggestedTicks = 3
		delta := 1
		for (max-min)/float64(delta) < float64(suggestedTicks) {
			delta += 10
		}
		ticks := make([]plot.Tick, 0)
		for i := int(min); i<= int(max); i+= delta{
			ticks = append(ticks, plot.Tick{Value:float64(i),Label:fmt.Sprintf("%d",i)})
		}
		return ticks
	}
	gt.X.Min=0
	gt.X.Max=float64(100)
	if err := gt.Save(5, 3, "gradeDistribution.png"); err != nil {
		panic(err)
	}

}
