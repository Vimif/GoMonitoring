package collectors

import "strings"

// isVirtualFS vérifie si le système de fichiers est virtuel
func isVirtualFS(fsType string) bool {
	virtualFS := []string{
		"tmpfs", "devtmpfs", "sysfs", "proc", "devpts",
		"cgroup", "cgroup2", "pstore", "securityfs",
		"debugfs", "hugetlbfs", "mqueue", "configfs",
		"fusectl", "binfmt_misc", "autofs", "overlay",
		"squashfs", "devfs",
	}

	fsLower := strings.ToLower(fsType)
	for _, vfs := range virtualFS {
		if fsLower == vfs {
			return true
		}
	}
	return false
}
