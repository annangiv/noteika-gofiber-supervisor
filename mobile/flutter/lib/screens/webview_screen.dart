import 'dart:io';
import 'package:flutter/material.dart';
import 'package:webview_flutter/webview_flutter.dart';
import 'package:webview_cookie_manager/webview_cookie_manager.dart';

class WebViewScreen extends StatefulWidget {
  const WebViewScreen({
    super.key,
    required this.url,
    required this.title,
  });

  final String url;
  final String title;

  @override
  State<WebViewScreen> createState() => _WebViewScreenState();
}

class _WebViewScreenState extends State<WebViewScreen> {
  late final WebViewController _controller;
  bool _loading = true;
  bool _finalizing = false;

  @override
  void initState() {
    super.initState();
    
    final userAgent = Platform.isAndroid
        ? 'Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Mobile Safari/537.36'
        : 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1';

    _controller = WebViewController()
      ..setJavaScriptMode(JavaScriptMode.unrestricted)
      ..setUserAgent(userAgent)
      ..setNavigationDelegate(
        NavigationDelegate(
          onNavigationRequest: (NavigationRequest request) {
            // Android emulator redirect bypass: replace localhost with 10.0.2.2
            if (Platform.isAndroid && request.url.startsWith('http://localhost:')) {
              final rewritten = request.url.replaceFirst('localhost', '10.0.2.2');
              _controller.loadRequest(Uri.parse(rewritten));
              return NavigationDecision.prevent;
            }
            if (request.url.contains('/notes') || request.url.contains('/dashboard')) {
              if (mounted) {
                setState(() {
                  _finalizing = true;
                });
              }
            }
            return NavigationDecision.navigate;
          },
          onPageStarted: (url) {
            final isRedirect = url.contains('/notes') || url.contains('/dashboard');
            if (isRedirect) {
              if (mounted) {
                setState(() {
                  _finalizing = true;
                });
              }
            } else {
              if (mounted) {
                setState(() {
                  _loading = true;
                });
              }
            }
          },
          onPageFinished: (url) async {
            final isRedirect = url.contains('/notes') || url.contains('/dashboard');
            if (isRedirect) {
              if (mounted) {
                setState(() {
                  _finalizing = true;
                });
              }
            } else {
              if (mounted) {
                setState(() {
                  _loading = false;
                });
              }
            }

            if (isRedirect) {
              try {
                final cookieManager = WebviewCookieManager();
                final gotCookies = await cookieManager.getCookies(url);
                String? sessionToken;
                for (final cookie in gotCookies) {
                  if (cookie.name == 'keller_session') {
                    sessionToken = cookie.value;
                    break;
                  }
                }

                if (sessionToken != null && mounted) {
                  Navigator.of(context).pop(sessionToken);
                }
              } catch (_) {}
            }
          },
        ),
      )
      ..loadRequest(Uri.parse(widget.url));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: const Color(0xFF0D1117),
      appBar: AppBar(
        title: Text(widget.title),
        backgroundColor: const Color(0xFF161B22),
        bottom: _loading && !_finalizing
            ? const PreferredSize(
                preferredSize: Size.fromHeight(2),
                child: LinearProgressIndicator(
                  backgroundColor: Colors.transparent,
                  color: Color(0xFF58A6FF),
                ),
              )
            : null,
      ),
      body: Stack(
        children: [
          WebViewWidget(controller: _controller),
          if (_finalizing)
            Container(
              color: const Color(0xFF0D1117),
              child: const Center(
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    CircularProgressIndicator(
                      color: Color(0xFF58A6FF),
                    ),
                    SizedBox(height: 16),
                    Text(
                      'Finalizing sign in...',
                      style: TextStyle(color: Colors.white, fontSize: 14),
                    ),
                  ],
                ),
              ),
            ),
        ],
      ),
    );
  }
}
