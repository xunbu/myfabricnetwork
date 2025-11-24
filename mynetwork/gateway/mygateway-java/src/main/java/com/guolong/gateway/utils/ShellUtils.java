package com.guolong.gateway.utils;

import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.util.List;
import java.util.Map;

public class ShellUtils {

    /**
     * 执行 Shell 命令
     * @param command 命令数组
     * @param env 环境变量 (可为 null)
     * @return 命令标准输出内容
     */
    public static String exec(List<String> command, Map<String, String> env) throws Exception {
        ProcessBuilder pb = new ProcessBuilder(command);
        
        // 设置环境变量
        if (env != null) {
            pb.environment().putAll(env);
        }
        
        // 合并标准错误流(stderr)到标准输出流(stdout)，方便调试报错
        pb.redirectErrorStream(true);

        Process process = pb.start();
        StringBuilder output = new StringBuilder();
        
        try (BufferedReader reader = new BufferedReader(new InputStreamReader(process.getInputStream()))) {
            String line;
            while ((line = reader.readLine()) != null) {
                output.append(line).append("\n");
            }
        }

        int exitCode = process.waitFor();
        if (exitCode != 0) {
            throw new RuntimeException("命令执行失败 (Exit Code " + exitCode + "):\n命令: " + String.join(" ", command) + "\n输出: " + output);
        }
        return output.toString();
    }
}