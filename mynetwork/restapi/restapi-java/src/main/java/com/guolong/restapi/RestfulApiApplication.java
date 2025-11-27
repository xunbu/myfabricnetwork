package com.guolong.restapi;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.scheduling.annotation.EnableScheduling;

@SpringBootApplication
@EnableScheduling
public class RestfulApiApplication {
    public static void main(String[] args) {
        SpringApplication.run(RestfulApiApplication.class, args);
    }
}