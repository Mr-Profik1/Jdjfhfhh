// LiteDroid: лёгкий запускатор Android-x86 9.0 на базе QEMU для слабых ПК (4 ГБ ОЗУ).
package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	isoURL      = "https://sourceforge.net/projects/android-x86/files/Release%209.0/android-x86_64-9.0-r2.iso/download"
	isoName     = "android-x86_64-9.0-r2.iso"
	diskName    = "android.qcow2"
	markerName  = "installed.flag"
	minISOBytes = 500 * 1024 * 1024
	defaultRAM  = "1024"
	defaultCPUs = "2"
	diskSize    = "10G"
)

func main() {
	exe, err := os.Executable()
	if err != nil {
		fail("Не удалось определить папку программы: %v", err)
	}
	base := filepath.Dir(exe)
	qemu := filepath.Join(base, "qemu", "qemu-system-x86_64.exe")
	qemuImg := filepath.Join(base, "qemu", "qemu-img.exe")
	dataDir := filepath.Join(base, "data")
	iso := filepath.Join(dataDir, isoName)
	// Свой образ (например, с Google Play) можно положить в папку data как custom.iso.
	if custom := filepath.Join(dataDir, "custom.iso"); fileExists(custom) {
		iso = custom
		fmt.Println("Найден свой образ custom.iso, использую его.")
	}
	disk := filepath.Join(dataDir, diskName)
	marker := filepath.Join(dataDir, markerName)

	for _, p := range []string{qemu, qemuImg} {
		if _, err := os.Stat(p); err != nil {
			fail("Не найден файл %s. Распакуйте весь архив целиком и не убирайте папку qemu.", p)
		}
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		fail("Не удалось создать папку data: %v", err)
	}

	installed := fileExists(marker)
	ramMB, cpuCores := loadConfig(filepath.Join(base, "litedroid.ini"))
	fmt.Printf("Память Android: %s МБ, ядер: %s (меняется в litedroid.ini)\n", ramMB, cpuCores)

	// Образ Android нужен только для первой установки.
	if !installed && !validISO(iso) {
		fmt.Println("Скачиваю образ Android-x86 9.0 (около 900 МБ), это один раз...")
		if err := download(isoURL, iso); err != nil {
			fail("Не удалось скачать образ: %v\nСкачайте %s вручную и положите в папку data.", err, isoName)
		}
	}

	if !fileExists(disk) {
		fmt.Println("Создаю виртуальный диск", diskSize, "(занимает место по мере использования)...")
		out, err := exec.Command(qemuImg, "create", "-f", "qcow2", disk, diskSize).CombinedOutput()
		if err != nil {
			fail("Не удалось создать диск: %v\n%s", err, out)
		}
	}

	args := []string{
		"-machine", "q35",
		// Ускорение через Windows Hypervisor Platform, если оно включено, иначе медленная программная эмуляция.
		"-accel", "whpx,kernel-irqchip=off",
		"-accel", "tcg,thread=multi,tb-size=128",
		"-cpu", "max",
		"-smp", cpuCores,
		"-m", ramMB,
		"-drive", "file=" + disk + ",format=qcow2",
		"-device", "virtio-balloon-pci", // возвращает неиспользуемую память Windows
		"-vga", "std",
		"-display", "gtk",
		"-device", "qemu-xhci",
		"-device", "usb-tablet",
		"-nic", "user,model=e1000",
		"-audiodev", "dsound,id=snd0",
		"-device", "intel-hda",
		"-device", "hda-duplex,audiodev=snd0",
		"-name", "LiteDroid",
	}

	if installed {
		args = append(args, "-boot", "c")
		fmt.Println("Запускаю Android...")
	} else {
		args = append(args, "-cdrom", iso, "-boot", "d")
		fmt.Println()
		fmt.Println("ПЕРВЫЙ ЗАПУСК: нужна установка Android на виртуальный диск.")
		fmt.Println("  1. В меню выберите Installation (стрелки и Enter).")
		fmt.Println("  2. Create/Modify partitions -> New -> Primary -> Write -> yes -> Quit.")
		fmt.Println("  3. Выберите созданный раздел, формат ext4, на вопросы GRUB и /system read-write ответьте Yes.")
		fmt.Println("  4. В конце выберите Run Android-x86 или закройте окно.")
		fmt.Println()
	}

	cmd := exec.Command(qemu, args...)
	cmd.Dir = filepath.Dir(qemu)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fail("QEMU завершился с ошибкой: %v", err)
	}

	if !installed {
		fmt.Print("Установка завершена успешно? Тогда в следующий раз Android запустится с диска (y/n): ")
		ans, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(ans)), "y") {
			_ = os.WriteFile(marker, []byte(time.Now().Format(time.RFC3339)), 0o644)
			fmt.Println("Готово. Запустите LiteDroid.exe снова.")
		}
	}
}

// loadConfig читает litedroid.ini (строки ram=... и cores=...), при отсутствии создаёт файл с минимальными значениями.
func loadConfig(path string) (string, string) {
	ram, cores := defaultRAM, defaultCPUs
	data, err := os.ReadFile(path)
	if err != nil {
		def := "# Память Android в МБ. 1024 это минимум для Android 9, на 4 ГБ ПК лучше не больше 1536.\nram=" + ram + "\n# Число ядер процессора.\ncores=" + cores + "\n"
		_ = os.WriteFile(path, []byte(def), 0o644)
		return ram, cores
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		k, v := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
		if v == "" || strings.Trim(v, "0123456789") != "" {
			continue
		}
		switch k {
		case "ram":
			ram = v
		case "cores":
			cores = v
		}
	}
	return ram, cores
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func validISO(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.Size() > minISOBytes
}

func download(url, dst string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	// SourceForge отдаёт прямой файл только "консольным" клиентам.
	req.Header.Set("User-Agent", "curl/8.4.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("сервер ответил: %s", resp.Status)
	}

	part := dst + ".part"
	f, err := os.Create(part)
	if err != nil {
		return err
	}
	var done int64
	total := resp.ContentLength
	buf := make([]byte, 256*1024)
	last := time.Now()
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				f.Close()
				return werr
			}
			done += int64(n)
			if time.Since(last) > time.Second {
				last = time.Now()
				if total > 0 {
					fmt.Printf("\r  %d%% (%d из %d МБ)", done*100/total, done>>20, total>>20)
				} else {
					fmt.Printf("\r  %d МБ", done>>20)
				}
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			f.Close()
			return rerr
		}
	}
	fmt.Println()
	if err := f.Close(); err != nil {
		return err
	}
	if done < minISOBytes {
		return fmt.Errorf("файл слишком маленький (%d МБ), скачивание не удалось", done>>20)
	}
	return os.Rename(part, dst)
}

func fail(format string, a ...any) {
	fmt.Printf("\nОШИБКА: "+format+"\n", a...)
	fmt.Print("Нажмите Enter, чтобы закрыть...")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	os.Exit(1)
}
