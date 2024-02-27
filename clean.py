import os
import subprocess

def rename_victim_files(root_dir):
    renamed_files = 0

    # 遍历指定根目录下的所有文件和文件夹
    for dirpath, dirnames, filenames in os.walk(root_dir):
        for filename in filenames:
            # 检查文件名是否以"victim"结尾
            if filename.endswith('-victim'):
                # 构造文件的完整路径
                file_path = os.path.join(dirpath, filename)

                # 获取新的文件名（去掉"victim"后缀）
                new_filename = filename[:-len('-victim')]

                # 构造新的文件完整路径
                new_file_path = os.path.join(dirpath, new_filename)

                # 执行 mv 命令来重命名文件
                subprocess.run(["mv", file_path, new_file_path])
                renamed_files += 1

    return renamed_files

if __name__ == "__main__":
    root_directory = "/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/"
    renamed_count = rename_victim_files(root_directory)

    print(f"共重命名了 {renamed_count} 个文件。")
