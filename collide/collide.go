package collide

import (
	"fmt"
	"log"
	"math/rand"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	STATE_SZ    = 729
	MSGBLOCK_SZ = 243
)

var (
	indices [STATE_SZ + 1]int
	box     = [11]int8{1, 0, -1, 0, 1, -1, 0, 0, -1, 1, 0}
)

type Result struct {
	found bool
	msg0  []int8
	msg1  []int8
}

// TODO: return ncollisions, not just the first one
func Collide(state []int8, template []int8, ncpu int, ncollisions, fail int) ([]int8, []int8, int64) {
	var wg sync.WaitGroup
	start := time.Now()
	total := make([]int64, ncpu)
	var collisions uint64
	var stop uint64
	results := make([]Result, ncpu)
	for i := 0; i < ncpu; i++ {
		wg.Add(1)
		go func(i int) {
			r := rand.New(rand.NewSource(int64(i) + time.Now().UnixNano()))
			var state0 [STATE_SZ]int8
			var state1 [STATE_SZ]int8
			found := false
			for {
				copy(state0[:], state)
				copy(state1[:], state)
				found = try_template(template, state0[:], state1[:], r, fail)
				if found {
					num_collisions := atomic.AddUint64(&collisions, 1)
					if num_collisions >= uint64(ncollisions) {
						res := Result{
							true,
							make([]int8, 243),
							make([]int8, 243),
						}
						copy(res.msg0, state0[0:243])
						copy(res.msg1, state1[0:243])
						results[i] = res
						break
					}
				}
				total[i]++
				should_stop := atomic.LoadUint64(&stop)
				if should_stop > 0 {
					// Someone else found collisions
					break
				}
			}
			atomic.AddUint64(&stop, 1)
			wg.Done()
		}(i)
	}
	wg.Wait()
	var sum int64
	for i := 0; i < ncpu; i++ {
		sum = sum + total[i]
	}
	found := false
	where := -1

	for i := 0; i < ncpu; i++ {
		if results[i].found == true {
			found = true
			where = i
			// log.Printf("Collision by goroutine %v.\nmsg0 = [%v]\nmsg1 = [%v]\n", i, Str(results[i].msg0), Str(results[i].msg1))
		}
	}
	log.Printf("%v checks in %v time\n", sum, time.Since(start))
	if found {
		return results[where].msg0, results[where].msg1, sum
	}
	fmt.Println("No collisions found")
	return nil, nil, sum
}

// Transform a slice of trits up to and including the specified message block (mb).
// This helps produce intermediate state to find collisions.
func ProduceState(trits []int8, mb int) []int8 {
	state := make([]int8, STATE_SZ)
	for i := 0; i <= mb; i++ {
		copy(state[0:MSGBLOCK_SZ], trits[i*MSGBLOCK_SZ:(i+1)*MSGBLOCK_SZ])
		transform(state)
	}
	return state
}

func exec_template(state, mb1, mb2, mb3 []int8, cmds string) string {
	cmd := cmds
	cmd = fmt.Sprintf("%v --state=[%v] --1mb=[%v] --2mb=[%v] --3mb=[%v]", cmd, Str(state), Str(mb1), Str(mb2), Str(mb3))
	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		fmt.Println(out)
		fmt.Printf("Panic on exec template %v\n", cmd)
		panic(err)
	}
	return string(out)
}

func RunPython(state, mb1, mb2, mb3 []int8, cmd string) ([]int8, []int8) {
	out := exec_template(state, mb1, mb2, mb3, cmd)
	re := regexp.MustCompile(`\[[(-1)\d\s,]*\]`)
	parts := re.FindAllStringSubmatch(out, -1)
	if len(parts) != 2 {
		fmt.Printf("%q\n", parts)
		panic("Bad parsing")
	}
	// log.Printf("Parts: %q\n", parts)
	tag_trits := make([]int8, 0)
	for _, c := range strings.Split(parts[0][0], ",") {
		i, err := strconv.Atoi(strings.Trim(c, "[] "))
		if err != nil {
			fmt.Printf("Panic on strconv\n")
			panic(err)
		}
		tag_trits = append(tag_trits, int8(i))
	}

	if len(tag_trits) != 243 {
		fmt.Println(len(tag_trits))
		panic("wrong length")
	}

	template_trits := make([]int8, 0)
	for _, c := range strings.Split(parts[1][0], ",") {
		i, err := strconv.Atoi(strings.Trim(c, "[] "))
		if err != nil {
			fmt.Printf("Panic on strconv\n")
			panic(err)
		}
		template_trits = append(template_trits, int8(i))
	}
	if len(template_trits) != 486 {
		fmt.Println(len(template_trits))
		panic("wrong length")
	}

	template_trits = template_trits[243:]
	return tag_trits, template_trits
}

func Str(a []int8) string {
	txt := []string{}
	for i := range a {
		number := a[i]
		text := strconv.Itoa(int(number))
		txt = append(txt, text)
	}
	return strings.Join(txt, ",")
}

func rand_trit(r *rand.Rand) int8 {
	x := int8(r.Intn(3))
	if x == 2 {
		x = -1
	}
	return x
}

// 1, 0, or -1: keep fixed
// 2: vary randomly
// 3: special trit, 0 or 1
func fill_template(template []int8, fill0 []int8, fill1 []int8, r *rand.Rand) int {
	specialTrit := -1
	for i := 0; i < len(template); i++ {
		if template[i] == 2 {
			x := rand_trit(r)
			fill0[i] = x
			fill1[i] = x
		} else if template[i] == 3 {
			if specialTrit >= 0 {
				panic("already set special trit index")
			}
			specialTrit = i
			fill0[i] = 0
			fill1[i] = 1
		} else {
			fill0[i] = template[i]
			fill1[i] = template[i]
		}
	}
	if specialTrit < 0 {
		panic("no special trit?")
	}
	return specialTrit
}

// transform does the transform from the sponge function
func transform(state []int8) {
	var cpy [STATE_SZ]int8
	for r := 0; r < 27; r++ {
		copy(cpy[:], state)
		for i := 0; i < STATE_SZ; i++ {
			state[i] = box[cpy[indices[i]]+(cpy[indices[i+1]]*4)+5]
		}
	}
}

// Run a transform on trits statea and stateb, compare result
func transform_cmp(statea, stateb [STATE_SZ]int8, fail int) bool {
	if len(statea) != STATE_SZ || len(stateb) != STATE_SZ {
		panic("bad state size")
	}
	var copya, copyb [STATE_SZ]int8

	for r := 0; r < 27; r++ {
		copy(copya[:], statea[:])
		copy(copyb[:], stateb[:])

		for i := 0; i < STATE_SZ; i++ {
			statea[i] = box[copya[indices[i]]+(copya[indices[i+1]]*4)+5]
			stateb[i] = box[copyb[indices[i]]+(copyb[indices[i+1]]*4)+5]
		}
		if fail > 0 && r == fail {
			diffs := 0
			diff0 := -1
			diff1 := -1
			for j := 0; j < STATE_SZ; j++ {
				if statea[j] != stateb[j] {
					diffs = diffs + 1
					if diff0 == -1 {
						diff0 = j
					} else {
						diff1 = j
					}
				}
			}
			if diffs > 1 {
				if r == fail {
					fmt.Printf("Number of diffs: %v, round %v\n", diffs, r)
					fmt.Printf("Diff: %v and %v trits \n", diff0, diff1)
					panic("Python code should have ensured that this doesn't happen")
				}
			}
		}
	}

	for i := 243; i < STATE_SZ; i++ {
		if statea[i] != stateb[i] {
			return false
		}
	}

	return true
}

func try_template(template []int8, state0, state1 []int8, r *rand.Rand, fail int) bool {
	fill_template(template, state0, state1, r)
	var state0tmp, state1tmp [STATE_SZ]int8
	copy(state0tmp[:], state0)
	copy(state1tmp[:], state1)
	worked := transform_cmp(state0tmp, state1tmp, fail)
	if !worked {
		return false
	}
	copy(state0tmp[:], state0)
	copy(state1tmp[:], state1)
	transform(state0tmp[:])
	transform(state1tmp[:])
	for i := 243; i < STATE_SZ; i++ {
		if state0tmp[i] != state1tmp[i] {
			panic("transform_cmp returned true but after transform last 2/3rds are different")
		}
	}
	return true
}

func init() {
	for i := 0; i < STATE_SZ; i++ {
		p := -365
		if indices[i] < 365 {
			p = 364
		}
		indices[i+1] = indices[i] + p
	}
}
