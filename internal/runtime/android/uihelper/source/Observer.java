package dev.agentenv.observer;

import android.app.Instrumentation;
import android.app.UiAutomation;
import android.accessibilityservice.AccessibilityServiceInfo;
import android.view.accessibility.AccessibilityNodeInfo;
import android.view.accessibility.AccessibilityWindowInfo;
import android.graphics.Rect;
import android.os.Bundle;
import android.os.SystemClock;
import android.util.Base64;
import org.json.*;
import java.util.*;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;

/** Private, bounded, self-targeting platform accessibility protocol. */
public final class Observer extends Instrumentation {
    private Bundle arguments;
    private boolean truncated;
    private int bytes;
    private boolean exhausted;
    private final List<AccessibilityNodeInfo> live = new ArrayList<>();
    private final List<JSONObject> values = new ArrayList<>();
    public void onCreate(Bundle args) { super.onCreate(args); arguments = args; start(); }
    private String field(CharSequence value) {
        String s = value == null ? "" : value.toString();
        if (s.length() > 4096) { truncated = true; return s.substring(0,4096); }
        return s;
    }
    private String digest(String value) throws Exception {
        byte[] hash = MessageDigest.getInstance("SHA-256").digest(value.getBytes(StandardCharsets.UTF_8));
        StringBuilder s = new StringBuilder(); for (byte b : hash) s.append(String.format(java.util.Locale.ROOT,"%02x",b & 255)); return s.toString();
    }
    private void node(AccessibilityNodeInfo n, String parent, String ancestry, int depth, JSONArray nodes) throws Exception {
        if (depth > 64) { truncated = true; return; }
        if (exhausted || live.size() >= 1000 || bytes >= 700000) { truncated = true; exhausted = true; return; }
        String ref = "n" + (live.size()+1); boolean secret = n.isEditable() || n.isPassword();
        Rect b = new Rect(); n.getBoundsInScreen(b);
        JSONObject j = new JSONObject();
        j.put("ref",ref); j.put("parent",parent); j.put("window_id",n.getWindowId());
        j.put("package",field(n.getPackageName())); j.put("class",field(n.getClassName()));
        j.put("resource_id",field(n.getViewIdResourceName())); j.put("text",secret ? "" : field(n.getText()));
        j.put("description",secret ? "" : field(n.getContentDescription())); j.put("hint",secret ? "" : field(n.getHintText()));
        j.put("bounds",new JSONArray(new int[]{b.left,b.top,b.right,b.bottom}));
        j.put("clickable",n.isClickable());j.put("long_clickable",n.isLongClickable());j.put("enabled",n.isEnabled());
        j.put("focusable",n.isFocusable());j.put("focused",n.isFocused());j.put("editable",n.isEditable());j.put("password",n.isPassword());
        j.put("scrollable",n.isScrollable());j.put("selected",n.isSelected());j.put("checkable",n.isCheckable());j.put("checked",n.isChecked());
        j.put("visible",n.isVisibleToUser());j.put("important",n.isImportantForAccessibility());
        List<Integer> ids = new ArrayList<>(); for(AccessibilityNodeInfo.AccessibilityAction a:n.getActionList()) ids.add(a.getId()); Collections.sort(ids);
        j.put("actions",new JSONArray(ids));
        // Fixed ordered array prevents JSONObject map-order dependence and ordinal identity.
        JSONArray identity = new JSONArray();identity.put(ancestry);
        for(String key:new String[]{"package","class","resource_id","text","description","hint","bounds","clickable","long_clickable","enabled","focusable","focused","editable","password","scrollable","selected","checkable","checked","visible","important","actions"}) identity.put(j.get(key));
        String fp=digest(identity.toString());j.put("fingerprint",fp);
        int size=j.toString().getBytes(StandardCharsets.UTF_8).length;
        if(bytes+size>700000){truncated=true;exhausted=true;return;}bytes+=size;live.add(n);values.add(j);nodes.put(j);
        for(int i=0;i<n.getChildCount();i++){
            if(exhausted || live.size()>=1000){truncated=true;exhausted=true;break;}
            AccessibilityNodeInfo c=n.getChild(i);if(c==null){truncated=true;continue;}
            node(c,ref,fp,depth+1,nodes);
        }
    }
    private JSONObject snapshot(UiAutomation automation,String pkg) throws Exception {
        live.clear(); values.clear(); truncated=false;bytes=0;exhausted=false;
        JSONArray windows=new JSONArray(),nodes=new JSONArray();
        for(AccessibilityWindowInfo w:automation.getWindows()){
            if(exhausted){truncated=true;break;}
            AccessibilityNodeInfo root=w.getRoot();if(root==null){truncated=true;continue;}
            if(!pkg.isEmpty()&&!pkg.equals(String.valueOf(root.getPackageName())))continue;
            if(windows.length()>=1000){truncated=true;break;}
            // Android window IDs can change on instrumentation reconnect even when
            // the visible window is unchanged. They are evidence, never identity.
            Rect bounds=new Rect();w.getBoundsInScreen(bounds);
            JSONArray windowIdentity=new JSONArray();windowIdentity.put(w.getType());
            windowIdentity.put(field(w.getTitle()));
            windowIdentity.put(new JSONArray(new int[]{bounds.left,bounds.top,bounds.right,bounds.bottom}));
            windowIdentity.put(field(root.getPackageName()));windowIdentity.put(field(root.getClassName()));windowIdentity.put(w.isActive());
            String windowKey=digest(windowIdentity.toString());
            windows.put(new JSONObject().put("id",w.getId()).put("key",windowKey).put("type",w.getType())
                    .put("title",field(w.getTitle()))
                    .put("bounds",new JSONArray(new int[]{bounds.left,bounds.top,bounds.right,bounds.bottom}))
                    .put("root_package",field(root.getPackageName()))
                    .put("root_class",field(root.getClassName()))
                    .put("active",w.isActive()));
            node(root,"","window:"+windowKey,0,nodes);
        }
        return new JSONObject().put("windows",windows).put("nodes",nodes).put("truncated",truncated);
    }
    public void onStart() {
        JSONObject out=new JSONObject();boolean dispatched=false;
        try {
            out.put("version",1);out.put("status","unavailable");out.put("action_performed",false);out.put("readback_equal",false);
            String encoded=arguments.getString("request","");if(encoded.length()>65536)throw new IllegalArgumentException();
            JSONObject req=new JSONObject(new String(Base64.decode(encoded,Base64.DEFAULT),StandardCharsets.UTF_8));
            if(req.getInt("version")!=1)throw new IllegalArgumentException();
            String op=req.getString("operation"), pkg=req.optString("package","");
            if(!op.equals("snapshot")&&!op.equals("tap")&&!op.equals("set-text"))throw new IllegalArgumentException();
            UiAutomation u=getUiAutomation();AccessibilityServiceInfo si=u.getServiceInfo();
            si.flags|=AccessibilityServiceInfo.FLAG_RETRIEVE_INTERACTIVE_WINDOWS|AccessibilityServiceInfo.FLAG_REPORT_VIEW_IDS|AccessibilityServiceInfo.FLAG_INCLUDE_NOT_IMPORTANT_VIEWS;
            u.setServiceInfo(si);SystemClock.sleep(1800);
            JSONObject observed=snapshot(u,pkg);out.put("snapshot",observed);
            if(op.equals("snapshot")){out.put("status","ok");}
            else if(truncated){out.put("status","refused");}
            else {
                String expected=req.getString("expected_fingerprint");int match=-1,count=0;
                for(int i=0;i<values.size();i++)if(expected.equals(values.get(i).getString("fingerprint"))){match=i;count++;}
                if(count==0)out.put("status","stale");else if(count>1)out.put("status","ambiguous");else{
                    AccessibilityNodeInfo target=live.get(match);int action=op.equals("tap")?AccessibilityNodeInfo.ACTION_CLICK:AccessibilityNodeInfo.ACTION_SET_TEXT;
                    boolean supported=false;for(AccessibilityNodeInfo.AccessibilityAction a:target.getActionList())if(a.getId()==action)supported=true;
                    if(!supported||!target.isEnabled()||!target.isVisibleToUser()||(op.equals("set-text")&&(!target.isEditable()||!target.isFocused())))out.put("status","refused");
                    else{
                        Bundle a=new Bundle();String text=req.optString("text","");if(text.length()>4096)throw new IllegalArgumentException();
                        if(op.equals("set-text"))a.putCharSequence(AccessibilityNodeInfo.ACTION_ARGUMENT_SET_TEXT_CHARSEQUENCE,text);
                        dispatched=true;out.put("action_performed",true);boolean accepted=target.performAction(action,a);
                        boolean equal=false;if(accepted&&op.equals("set-text")){
                            long deadline=SystemClock.uptimeMillis()+1000;
                            do {
                                equal=target.refresh()&&text.contentEquals(target.getText()==null?"":target.getText());
                                if(equal)break;
                                long remaining=deadline-SystemClock.uptimeMillis();if(remaining<=0)break;SystemClock.sleep(Math.min(100,remaining));
                            } while(true);
                            out.put("readback_equal",equal);
                        }
                        out.put("status",accepted&&(op.equals("tap")||equal)?"ok":"uncertain");
                        JSONObject after=snapshot(u,pkg);JSONArray state=new JSONArray();
                        JSONArray afterWindows=after.getJSONArray("windows");
                        for(int wi=0;wi<afterWindows.length();wi++)state.put(afterWindows.getJSONObject(wi).getString("key"));
                        state.put(after.getBoolean("truncated"));
                        // Each node fingerprint already excludes ordinals and sensitive values.
                        for(JSONObject value:values)state.put(value.getString("fingerprint"));
                        out.put("after_fingerprint",digest(state.toString()));
                    }
                }
            }
        }catch(Throwable failure){try{out.put("status",dispatched?"uncertain":"unavailable");}catch(Exception ignored){}}
        String json=out.toString();if(json.getBytes(StandardCharsets.UTF_8).length>1048576)json="{\"version\":1,\"status\":\"unavailable\"}";
        Bundle result=new Bundle();result.putString("response64",Base64.encodeToString(json.getBytes(StandardCharsets.UTF_8),Base64.NO_WRAP));finish(0,result);
    }
}
