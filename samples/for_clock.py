import time, sys

firsttick = True
print()
for hour in range(0,23) {
    for minute in range(0,60) {
        for second in range(0,60) {
            if (firsttick == False) {
                sys.stdout.write('\033[F')
                } # Move cursor up one line
            time.sleep(1)
            print(f"{hour}:{minute}:{second}")
            firsttick = False
            }
        }
    }