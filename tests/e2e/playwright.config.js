import {defineConfig} from '@playwright/test';
export default defineConfig({
 testDir:'.',testMatch:'*.spec.js',workers:1,retries:0,timeout:30000,
 reporter:[['line'],['json',{outputFile:'/results/resultados.json'}]],
 outputDir:'/results/artefactos',
 use:{baseURL:process.env.BASE_URL||'http://localhost:8080',trace:'retain-on-failure',screenshot:'only-on-failure'},
 projects:[{name:'escritorio',use:{viewport:{width:1280,height:900}}},{name:'movil',use:{viewport:{width:390,height:844}}}],
});
