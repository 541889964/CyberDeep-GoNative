package com.cyber.deep;

import android.os.Bundle;
import android.text.TextUtils;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.*;
import androidx.annotation.NonNull;
import androidx.appcompat.app.AppCompatActivity;
import androidx.recyclerview.widget.LinearLayoutManager;
import androidx.recyclerview.widget.RecyclerView;
import core.Core;
import core.StreamListener;
import java.util.ArrayList;
import java.util.List;

public class MainActivity extends AppCompatActivity {
    private RecyclerView recyclerView;
    private EditText etInput;
    private TextView tvStatus;
    private boolean isRunning = false;
    private final String TOKEN = "WlS6knRr+HvB+h7ioKh4LgxmyfJXJfQ3jJaYfuYzQQ9oOthcRovka6o83lKFspw5";

    private MsgAdapter adapter;
    private final List<Msg> msgList = new ArrayList<>();

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);

        recyclerView = findViewById(R.id.recyclerView);
        etInput = findViewById(R.id.etInput);
        ImageButton btnSend = findViewById(R.id.btnSend);
        tvStatus = findViewById(R.id.tvStatus);

        adapter = new MsgAdapter();
        recyclerView.setLayoutManager(new LinearLayoutManager(this));
        recyclerView.setAdapter(adapter);

        btnSend.setOnClickListener(v -> {
            String prompt = etInput.getText().toString().trim();
            if (TextUtils.isEmpty(prompt) || isRunning) return;
            etInput.setText("");
            addMsg("User", prompt);
            isRunning = true;
            tvStatus.setText("生成中...");

            Core.sendMessage(TOKEN, prompt, new StreamListener() {
                @Override
                public void onChunk(String text) {
                    runOnUiThread(() -> appendAiMsg(text));
                }
                @Override
                public void onFinish() {
                    runOnUiThread(() -> {
                        isRunning = false;
                        tvStatus.setText("就绪");
                    });
                }
                @Override
                public void onError(String errMsg) {
                    runOnUiThread(() -> {
                        addMsg("System", "❌ 错误: " + errMsg);
                        isRunning = false;
                        tvStatus.setText("就绪");
                    });
                }
            });
        });
    }

    private void appendAiMsg(String text) {
        if (msgList.isEmpty() || !"AI".equals(msgList.get(msgList.size() - 1).role)) {
            addMsg("AI", text);
        } else {
            Msg last = msgList.get(msgList.size() - 1);
            last.content += text;
            adapter.notifyItemChanged(msgList.size() - 1);
            recyclerView.smoothScrollToPosition(msgList.size() - 1);
        }
    }

    private void addMsg(String role, String content) {
        msgList.add(new Msg(role, content));
        adapter.notifyItemInserted(msgList.size() - 1);
        recyclerView.smoothScrollToPosition(msgList.size() - 1);
    }

    class MsgAdapter extends RecyclerView.Adapter<MsgAdapter.VH> {
        @NonNull @Override public VH onCreateViewHolder(@NonNull ViewGroup parent, int viewType) {
            return new VH(LayoutInflater.from(parent.getContext()).inflate(R.layout.item_message, parent, false));
        }
        @Override public void onBindViewHolder(@NonNull VH holder, int position) {
            Msg msg = msgList.get(position);
            holder.tvContent.setText(msg.content);
            LinearLayout.LayoutParams params = new LinearLayout.LayoutParams(ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT);
            if ("User".equals(msg.role)) {
                params.gravity = android.view.Gravity.END;
                holder.tvContent.setBackgroundResource(R.drawable.bg_user_bubble);
                holder.tvContent.setTextColor(0xFFFFFFFF);
            } else if ("System".equals(msg.role)) {
                params.gravity = android.view.Gravity.CENTER;
                holder.tvContent.setBackgroundResource(R.drawable.bg_ai_bubble);
                holder.tvContent.setTextColor(0xFFEF4444);
            } else {
                params.gravity = android.view.Gravity.START;
                holder.tvContent.setBackgroundResource(R.drawable.bg_ai_bubble);
                holder.tvContent.setTextColor(0xFF00E5FF);
            }
            holder.tvContent.setLayoutParams(params);
        }
        @Override public int getItemCount() { return msgList.size(); }
        class VH extends RecyclerView.ViewHolder { TextView tvContent; VH(View v) { super(v); tvContent = v.findViewById(R.id.tvContent); } }
    }
    static class Msg { String role, content; Msg(String role, String content) { this.role = role; this.content = content; } }
}
