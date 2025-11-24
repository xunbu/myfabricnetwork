package com.guolong.gateway.utils;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.stream.Stream;

public class FileUtils {
    public static Path getFirstFile(String dirPath) throws IOException {
        try (Stream<Path> stream = Files.list(Paths.get(dirPath))) {
            return stream.findFirst()
                    .orElseThrow(() -> new IOException("目录中没有文件: " + dirPath));
        }
    }
}