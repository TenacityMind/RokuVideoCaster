package com.rokucaster.app;

import android.app.Activity;
import android.content.Intent;
import android.net.Uri;
import android.os.Bundle;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.webkit.WebViewClient;
import android.widget.Toast;

public class MainActivity extends Activity {
    private WebView webView;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);

        webView = new WebView(this);
        setContentView(webView);

        WebSettings settings = webView.getSettings();
        settings.setJavaScriptEnabled(true);
        settings.setDomStorageEnabled(true);
        settings.setAllowFileAccess(true);
        settings.setMixedContentMode(WebSettings.MIXED_CONTENT_ALWAYS_ALLOW);

        webView.setWebViewClient(new WebViewClient());

        // Handle shared URL from other apps
        Intent intent = getIntent();
        String action = intent.getAction();
        String type = intent.getType();

        String targetUrl = "http://192.168.1.100:8080"; // Default local server fallback or configurable in settings

        if (Intent.ACTION_SEND.equals(action) && type != null) {
            if ("text/plain".equals(type)) {
                String sharedText = intent.getStringExtra(Intent.EXTRA_TEXT);
                if (sharedText != null) {
                    Toast.makeText(this, "Sharing video to Roku TV...", Toast.LENGTH_SHORT).show();
                    targetUrl += "/#webStreamUrl=" + Uri.encode(sharedText);
                }
            }
        }

        webView.loadUrl(targetUrl);
    }
}
