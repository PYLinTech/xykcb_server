package cn.pylin.xykcb;

import android.content.Context;
import android.content.SharedPreferences;
import android.os.Handler;
import android.os.Looper;
import android.util.Base64;
import org.json.JSONArray;
import org.json.JSONException;
import org.json.JSONObject;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Calendar;
import java.util.HashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.concurrent.TimeUnit;
import javax.crypto.Cipher;
import javax.crypto.spec.SecretKeySpec;
import okhttp3.Call;
import okhttp3.Callback;
import okhttp3.Cookie;
import okhttp3.CookieJar;
import okhttp3.FormBody;
import okhttp3.HttpUrl;
import okhttp3.OkHttpClient;
import okhttp3.Request;
import okhttp3.RequestBody;
import okhttp3.Response;
import okhttp3.ResponseBody;

/**
 * 登录管理器类 - 合并了所有学校的登录逻辑
 */
public class LoginManager {
    private final Context context;
    private final CourseDataCallback callback;
    private OkHttpClient httpClient;
    private String currentWeek = "1";
    private String schoolCode;
    // 标志变量：记录是否已经尝试过网络更新
    private boolean hasAttemptedUpdate = false;
    // 标志变量：记录是否已经显示过首次获取课程的提示
    private boolean hasShownFirstCoursePrompt = false;
    
    // 运行时变量：存储登录用户信息
    private String runtimeUserName = "";
    private String runtimeAcademyName = "";
    private String runtimeClassName = "";

    // 登录类型枚举
    public enum LoginType {
        HNIT_A(1, "HNIT-A", "湖南工学院(通用)"),
        HNIT_B(2, "HNIT-B", "湖南工学院(内网)"),
        HYNU(3, "HYNU", "衡阳师范学院"),
        USC(4, "USC", "南华大学");

        private final int id;
        private final String code;
        private final String name;

        LoginType(int id, String code, String name) {
            this.id = id;
            this.code = code;
            this.name = name;
        }

        public int getId() {
            return id;
        }

        public String getCode() {
            return code;
        }

        public String getName() {
            return name;
        }

        public static LoginType fromCode(String code) {
            for (LoginType type : values()) {
                if (type.code.equals(code)) {
                    return type;
                }
            }
            return HNIT_A; // 默认返回湖南工学院
        }
    }

    public interface CourseDataCallback {
        void onCourseDataReceived(List<List<Course>> weeklyCourses);
        void onError(String message);
    }
    
    /**
     * 获取运行时用户信息
     */
    public String getRuntimeUserName() {
        return runtimeUserName;
    }
    
    public String getRuntimeAcademyName() {
        return runtimeAcademyName;
    }
    
    public String getRuntimeClassName() {
        return runtimeClassName;
    }

    public LoginManager(Context context, CourseDataCallback callback) {
        this.context = context;
        this.callback = callback;
    }

    /**
     * 执行登录操作
     * @param username 用户名
     * @param password 密码
     * @param schoolCode 学校代码
     */
    public void performLogin(String username, String password, String schoolCode) {
        if (username == null || username.isEmpty() || password == null || password.isEmpty()) {
            notifyError("用户名或密码不能为空");
            return;
        }

        // 重置更新标志，确保每次登录都是一个新的会话状态
        hasAttemptedUpdate = false;
        // 重置首次获取课程提示标志，确保每次应用运行时都能显示提示
        hasShownFirstCoursePrompt = false;
        
        this.schoolCode = schoolCode;
        LoginType loginType = LoginType.fromCode(schoolCode);

        // 首先检查是否存在本地数据，如果不存在，先显示加载提示
        SharedPreferences sharedPreferences = context.getSharedPreferences("CourseListInfo", Context.MODE_PRIVATE);
        String localCourseList = sharedPreferences.getString("CourseList", "");
        
        if (localCourseList.isEmpty()) {
            notifyError("正在获取课程数据...");
        } else {
            // 存在本地数据，先加载本地数据
            loadLocalCourseData();
        }

        // 然后根据学校类型执行不同的登录逻辑
        switch (loginType) {
            case HNIT_A:
                performHnitALogin(username, password);
                break;
//            case HNIT_B:
//                performHnitBLogin(username, password);
//                break;
            // 可以在这里添加其他学校的登录逻辑
            default:
                notifyError("暂不支持该学校，敬请期待");
        }
    }

    /**
     * 湖南工学院外网登录
     */
    private void performHnitALogin(String username, String password) {
        // 初始化 OkHttpClient
        httpClient = new OkHttpClient.Builder()
                .connectTimeout(10, TimeUnit.SECONDS)
                .readTimeout(10, TimeUnit.SECONDS)
                .writeTimeout(10, TimeUnit.SECONDS)
                .build();

        String encryptedPassword = encryptPassword(password);
        String loginUrl = "https://jw.hnit.edu.cn/njwhd/login?userNo=" + username + "&pwd=" + encryptedPassword;

        Request request = new Request.Builder()
                .url(loginUrl)
                .post(RequestBody.create(new byte[0]))
                .build();

        httpClient.newCall(request).enqueue(new Callback() {
            @Override
            public void onFailure(Call call, IOException e) {
                notifyError("登录失败");
            }

            @Override
            public void onResponse(Call call, Response response) throws IOException {
                if (response.isSuccessful()) {
                    try (ResponseBody responseBody = response.body()) {
                        if (responseBody != null) {
                            String responseString = responseBody.string();
                            JSONObject jsonResponse = new JSONObject(responseString);
                            
                            // 获取Msg字段内容
                            String msg = jsonResponse.optString("Msg", "");
                            
                            // 根据Msg内容判断登录结果
                            if (msg.contains("成功")) {
                                // 登录成功，检查是否有token
                                if (jsonResponse.has("data") && jsonResponse.getJSONObject("data").has("token")) {
                                    JSONObject data = jsonResponse.getJSONObject("data");
                                    String token = data.getString("token");
                                    saveLoginInfo(username, password);
                                    // 保存token用于后续获取周次
                                    SharedPreferences sharedPreferences = context.getSharedPreferences("LoginInfo", Context.MODE_PRIVATE);
                                    SharedPreferences.Editor editor = sharedPreferences.edit();
                                    editor.putString("token", token);
                                    
                                    // 保存用户信息到运行时变量
                                    if (data.has("name")) {
                                        runtimeUserName = data.getString("name");
                                    }
                                    if (data.has("academyName")) {
                                        runtimeAcademyName = data.getString("academyName");
                                    }
                                    if (data.has("clsName")) {
                                        runtimeClassName = data.getString("clsName");
                                    }
                                    editor.apply();
                                    
                                    getXnxqListId(token);
                                } else {
                                    notifyError("登录失败：服务器返回异常");
                                }
                            } else if (msg.contains("错误")) {
                                // 账号或密码错误
                                notifyError("登录失败：该帐号不存在或密码错误");
                            } else {
                                // 其他未知情况
                                notifyError("登录失败：服务器返回异常");
                            }
                        }
                    } catch (JSONException e) {
                        notifyError("登录失败：数据解析错误");
                    }
                } else {
                    notifyError("登录失败：服务器错误");
                }
            }
        });
    }

    void loadLocalCourseData() {
        try {
            SharedPreferences sharedPreferences = context.getSharedPreferences("CourseListInfo", Context.MODE_PRIVATE);
            String localCourseList = sharedPreferences.getString("CourseList", "");

            if (!localCourseList.isEmpty()) {
                // 检查是否有保存的真实周次信息
                SharedPreferences weekPrefs = context.getSharedPreferences("WeekDates", Context.MODE_PRIVATE);
                if (weekPrefs.getAll().isEmpty()) {
                    // 如果没有周次信息，尝试从网络获取真实周次
                    SharedPreferences loginPrefs = context.getSharedPreferences("LoginInfo", Context.MODE_PRIVATE);
                    String token = loginPrefs.getString("token", "");
                    if (!token.isEmpty()) {
                        getCurrentWeekFromApi(token);
                        return; // 等待网络获取完成后再加载数据
                    }
                }
                
                // 使用现有的周次信息
                int currentWeek = CourseDataManager.getCurrentWeek(context);
                // 更新MainActivity中的Week静态变量，确保UI显示正确的周次
                MainActivity.Week = String.valueOf(currentWeek);
                
                List<List<Course>> weeklyCourses = CourseDataManager.parseCourseData(localCourseList);
                callback.onCourseDataReceived(weeklyCourses);
            } else {
                notifyError("正在尝试更新数据...");
            }
        } catch (Exception e) {
            notifyError("加载本地数据失败");
        }
    }

    /**
     * 加密密码（湖南工学院外网登录使用）
     */
    private String encryptPassword(String password) {
        try {
            byte[] keyBytes = hexStringToByteArray("717a6b6a316b6a6768643d383736262a");
            Cipher cipher = Cipher.getInstance("AES/ECB/PKCS5Padding");
            SecretKeySpec keySpec = new SecretKeySpec(keyBytes, "AES");
            cipher.init(Cipher.ENCRYPT_MODE, keySpec);

            String quotedPassword = "\"" + password + "\"";
            byte[] encryptedBytes = cipher.doFinal(quotedPassword.getBytes(StandardCharsets.UTF_8));

            String base64Encoded = Base64.encodeToString(encryptedBytes, Base64.NO_WRAP);
            String doubleBase64Encoded = Base64.encodeToString(base64Encoded.getBytes(StandardCharsets.UTF_8), Base64.NO_WRAP);

            return doubleBase64Encoded;
        } catch (Exception e) {
            return null;
        }
    }

    /**
     * 十六进制字符串转字节数组
     */
    private byte[] hexStringToByteArray(String s) {
        int len = s.length();
        byte[] data = new byte[len / 2];
        for (int i = 0; i < len; i += 2) {
            data[i / 2] = (byte) ((Character.digit(s.charAt(i), 16) << 4)
                    + Character.digit(s.charAt(i + 1), 16));
        }
        return data;
    }

    /**
     * 保存登录信息
     */
    private void saveLoginInfo(String username, String password) {
        SharedPreferences sharedPreferences = context.getSharedPreferences("LoginInfo", Context.MODE_PRIVATE);
        SharedPreferences.Editor editor = sharedPreferences.edit();
        editor.putString("username", username);
        editor.putString("password", password);
        editor.putString("schoolCode", schoolCode);
        editor.apply();
        
        // 保存到多账号列表
        MultiAccountManager accountManager = new MultiAccountManager(context);
        String schoolName = getSchoolNameByCode(schoolCode);
        accountManager.saveAccount(username, password, schoolCode, schoolName);
    }
    
    /**
     * 根据学校代码获取学校名称
     */
    private String getSchoolNameByCode(String schoolCode) {
        for (LoginType loginType : LoginType.values()) {
            if (loginType.getCode().equals(schoolCode)) {
                return loginType.getName();
            }
        }
        return "未知学校";
    }

    /**
     * 获取学年学期ID（湖南工学院外网登录使用）
     */
    private void getXnxqListId(String token) {
        String XnxqListIdUrl = "https://jw.hnit.edu.cn/njwhd/getXnxqList?token=" + token;

        Request request = new Request.Builder()
                .url(XnxqListIdUrl)
                .post(RequestBody.create(new byte[0]))
                .build();

        httpClient.newCall(request).enqueue(new Callback() {
            @Override
            public void onFailure(Call call, IOException e) {
                notifyError("获取学年学期失败：" + e.getMessage());
            }

            @Override
            public void onResponse(Call call, Response response) throws IOException {
                if (response.isSuccessful()) {
                    try (ResponseBody responseBody = response.body()) {
                        if (responseBody != null) {
                            String responseString = responseBody.string();
                            JSONArray jsonArray = new JSONArray(responseString);
                            JSONObject maxNumObject = null;
                            int maxNum = Integer.MIN_VALUE;

                            for (int i = 0; i < jsonArray.length(); i++) {
                                JSONObject jsonObject = jsonArray.getJSONObject(i);
                                int num = jsonObject.getInt("num");
                                if (num > maxNum) {
                                    maxNum = num;
                                    maxNumObject = jsonObject;
                                }
                            }

                            if (maxNumObject != null) {
                                String xnxq01id = maxNumObject.getString("xnxq01id");
                                // 先获取当前周次，然后在回调中获取课程列表
                                getCurrentWeekFromApi(token);
                                // 同时获取课程节次模式
                                getKbjcmsid(token, xnxq01id);
                            } else {
                                notifyError("获取学年学期失败！");
                            }
                        }
                    } catch (JSONException e) {
                        notifyError("解析学年学期数据失败：" + e.getMessage());
                    }
                } else {
                    notifyError("获取学年学期失败，错误码：" + response.code());
                }
            }
        });
    }

    private void getCurrentWeekFromApi(String token) {
        String teachingWeekUrl = "https://jw.hnit.edu.cn/njwhd/teachingWeek?token=" + token;

        Request request = new Request.Builder()
                .url(teachingWeekUrl)
                .post(RequestBody.create(new byte[0]))
                .build();

        httpClient.newCall(request).enqueue(new Callback() {
            @Override
            public void onFailure(Call call, IOException e) {
                notifyError("获取当前周次失败：" + e.getMessage());
                currentWeek = "1";
            }

            @Override
            public void onResponse(Call call, Response response) throws IOException {
                if (response.isSuccessful()) {
                    try (ResponseBody responseBody = response.body()) {
                        if (responseBody != null) {
                            String responseString = responseBody.string();
                            JSONObject jsonResponse = new JSONObject(responseString);
                            if (jsonResponse.has("nowWeek")) {
                                String week = jsonResponse.getString("nowWeek");
                                currentWeek = week;
                                // 计算并保存每周的日期范围
                                calculateAndSaveWeekDates(Integer.parseInt(currentWeek));
                                
                                // 保存当前周次到SharedPreferences
                                SharedPreferences prefs = context.getSharedPreferences("LoginInfo", Context.MODE_PRIVATE);
                                prefs.edit().putString("currentWeek", currentWeek).apply();

                            } else {
                                notifyError("获取当前周次失败：服务器返回数据不完整");
                                currentWeek = "1";
                            }
                        }
                    } catch (JSONException e) {
                        notifyError("解析周次数据失败：" + e.getMessage());
                        currentWeek = "1";
                    }
                } else {
                    notifyError("获取当前周次失败，错误码：" + response.code());
                    currentWeek = "1";
                }
            }
        });
    }

    /**
     * 计算并保存每周的日期范围
     * @param currentWeek 当前周数
     */
    private void calculateAndSaveWeekDates(int currentWeek) {
        try {
            // 获取当前系统日期
            Calendar calendar = Calendar.getInstance();
            int todayOfWeek = calendar.get(Calendar.DAY_OF_WEEK);
            
            // 计算本周周一的日期（Calendar中周日是1，周一是2，所以需要调整）
            int daysToMonday = (todayOfWeek == Calendar.SUNDAY) ? -6 : 2 - todayOfWeek;
            calendar.add(Calendar.DAY_OF_MONTH, daysToMonday);
            
            // 计算第1周周一的日期
            calendar.add(Calendar.DAY_OF_MONTH, -(currentWeek - 1) * 7);
            Calendar firstWeekMonday = (Calendar) calendar.clone();
            
            // 保存每周的日期范围
            SharedPreferences weekDatesPrefs = context.getSharedPreferences("WeekDates", Context.MODE_PRIVATE);
            SharedPreferences.Editor editor = weekDatesPrefs.edit();
            editor.clear(); // 清除旧数据
            
            SimpleDateFormat sdf = new SimpleDateFormat("M.d", Locale.getDefault());
            
            // 计算并保存每周的日期范围（支持1-24周）
            for (int week = 1; week <= 24; week++) {
                Calendar weekStart = (Calendar) firstWeekMonday.clone();
                weekStart.add(Calendar.DAY_OF_MONTH, (week - 1) * 7);
                Calendar weekEnd = (Calendar) weekStart.clone();
                weekEnd.add(Calendar.DAY_OF_MONTH, 6);
                
                StringBuilder dateRange = new StringBuilder();
                for (int day = 0; day < 7; day++) {
                    Calendar currentDay = (Calendar) weekStart.clone();
                    currentDay.add(Calendar.DAY_OF_MONTH, day);
                    dateRange.append(sdf.format(currentDay.getTime()));
                    if (day < 6) {
                        dateRange.append(",");
                    }
                }
                
                editor.putString(String.valueOf(week), dateRange.toString());
            }
            
            editor.apply();
        } catch (Exception e) {
        }
    }

    /**
     * 获取课程节次模式（湖南工学院外网登录使用）
     */
    private void getKbjcmsid(String token, String xnxq01id) {
        String sjkbmsUrl = "https://jw.hnit.edu.cn/njwhd/Get_sjkbms?token=" + token;

        Request request = new Request.Builder()
                .url(sjkbmsUrl)
                .post(RequestBody.create(new byte[0]))
                .build();

        httpClient.newCall(request).enqueue(new Callback() {
            @Override
            public void onFailure(Call call, IOException e) {
                notifyError("获取课程节次模式失败：" + e.getMessage());
            }

            @Override
            public void onResponse(Call call, Response response) throws IOException {
                if (response.isSuccessful()) {
                    try (ResponseBody responseBody = response.body()) {
                        if (responseBody != null) {
                            String responseString = responseBody.string();
                            JSONObject jsonResponse = new JSONObject(responseString);
                            if (jsonResponse.has("data") && jsonResponse.getJSONArray("data").length() > 0) {
                                String kbjcmsid = jsonResponse.getJSONArray("data").getJSONObject(0).getString("kbjcmsid");
                                getCourseList(token, xnxq01id, kbjcmsid);
                            } else {
                                notifyError("获取课程节次模式失败！");
                            }
                        }
                    } catch (JSONException e) {
                        notifyError("解析课程节次模式数据失败：" + e.getMessage());
                    }
                } else {
                    notifyError("获取课程节次模式失败，错误码：" + response.code());
                }
            }
        });
    }

    /**
     * 获取课程列表（湖南工学院外网登录使用）
     */
    private void getCourseList(String token, String xnxq01id, String kbjcmsid) {
        String CourseListUrl = "https://jw.hnit.edu.cn/njwhd/student/curriculum?token=" + token +
                "&xnxq01id=" + xnxq01id + "&kbjcmsid=" + kbjcmsid + "&week=all";

        Request request = new Request.Builder()
                .url(CourseListUrl)
                .post(RequestBody.create(new byte[0]))
                .build();

        httpClient.newCall(request).enqueue(new Callback() {
            @Override
            public void onFailure(Call call, IOException e) {
                notifyError("获取课程列表失败：" + e.getMessage());
            }

            @Override
            public void onResponse(Call call, Response response) throws IOException {
                if (response.isSuccessful()) {
                    try (ResponseBody responseBody = response.body()) {
                        if (responseBody != null) {
                            String newCourseList = responseBody.string();

                            SharedPreferences sharedPreferences = context.getSharedPreferences("CourseListInfo", Context.MODE_PRIVATE);
                            String existingCourseList = sharedPreferences.getString("CourseList", "");

                            // 设置标志表示已经尝试过网络更新
                            hasAttemptedUpdate = true;
                            
                            // 直接保存新的课程数据，不进行比对
                            sharedPreferences.edit().putString("CourseList", newCourseList).apply();

                            List<List<Course>> weeklyCourses = CourseDataManager.parseCourseData(newCourseList);
                            // 设置当前周次
                            SharedPreferences loginPrefs = context.getSharedPreferences("LoginInfo", Context.MODE_PRIVATE);
                            String token = loginPrefs.getString("token", "");
                            if (!token.isEmpty()) {
                                getCurrentWeekFromApi(token);
                            }
                            
                            // 如果是应用运行后第一次成功获取课程数据，显示提示
                            if (!hasShownFirstCoursePrompt) {
                                hasShownFirstCoursePrompt = true;
                                showFirstCoursePrompt();
                            }
                            
                            callback.onCourseDataReceived(weeklyCourses);
                        }
                    } catch (Exception e) {
                        notifyError("解析课程列表数据失败：" + e.getMessage());
                    }
                } else {
                    notifyError("获取课程列表失败，错误码：" + response.code());
                }
            }
        });
    }


    
    /**
     * 从系统获取当前周次（湖南工学院内网登录使用）
     */
    private void getCurrentWeekFromSystem() throws IOException {
        Request weekRequest = new Request.Builder()
                .url("https://jwxt.hnit.edu.cn/jsxsd/xskb/xskb_list.do")
                .build();

        try (Response weekResponse = httpClient.newCall(weekRequest).execute()) {
            if (!weekResponse.isSuccessful()) {
                throw new IOException("获取当前周次失败: " + weekResponse.code());
            }

            String html = weekResponse.body().string();
            int startIndex = html.indexOf("var zc = ");
            if (startIndex != -1) {
                int endIndex = html.indexOf(";" , startIndex);
                if (endIndex != -1) {
                    String weekStr = html.substring(startIndex + 9, endIndex).trim();
                    currentWeek = weekStr;
                }
            }
        }
    }

    private void notifyError(String message) {
        new Handler(Looper.getMainLooper()).post(() -> {
            callback.onError(message);
        });
    }
    
    /**
     * 显示首次获取课程数据的提示
     */
    private void showFirstCoursePrompt() {
        new Handler(Looper.getMainLooper()).post(() -> {
            CustomToast.showShortToast(context, "当前是最新课程数据");
        });
    }
}