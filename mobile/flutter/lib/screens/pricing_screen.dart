import 'dart:io';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:url_launcher/url_launcher.dart';

import '../state/app_state.dart';

class PricingScreen extends StatefulWidget {
  const PricingScreen({super.key});

  @override
  State<PricingScreen> createState() => _PricingScreenState();
}

class _PricingScreenState extends State<PricingScreen> {
  bool _loading = false;
  String? _error;

  Future<void> _upgrade() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final state = context.read<AppState>();

      if (!kIsWeb && (Platform.isAndroid || Platform.isIOS)) {
        debugPrint('PricingScreen: Initiating native mobile IAP purchase...');
        await state.iapService.buyProSubscription();
        if (state.iapService.errorMessage != null) {
          throw Exception(state.iapService.errorMessage);
        }
        return;
      }

      final res = await state.api.checkoutStripe();
      final urlStr = res['url'] as String?;
      if (urlStr != null) {
        final url = Uri.parse(urlStr);
        if (await canLaunchUrl(url)) {
          await launchUrl(url, mode: LaunchMode.externalApplication);
        } else {
          throw Exception('Could not launch Stripe Checkout URL.');
        }
      } else {
        throw Exception('Server failed to return Checkout URL.');
      }
    } catch (e) {
      setState(() => _error = e.toString());
    } finally {
      setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final state = context.watch<AppState>();
    final isPro = state.user?['pro_access'] == true;
    final isIapPending = !kIsWeb && state.iapService.purchasePending;

    return Scaffold(
      appBar: AppBar(
        title: const Text('Pricing & Plans'),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back_ios_new, size: 18),
          onPressed: () => Navigator.pop(context),
        ),
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const Text(
              'PLANS',
              style: TextStyle(
                color: Color(0xFFA78BFA),
                fontWeight: FontWeight.w600,
                letterSpacing: 1.5,
                fontSize: 12,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 8),
            const Text(
              'Simple, honest pricing',
              style: TextStyle(
                color: Colors.white,
                fontWeight: FontWeight.bold,
                fontSize: 28,
                letterSpacing: -0.5,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 12),
            const Text(
              'Start free with 10 notes. Upgrade when Noteika becomes part of your daily workflow.',
              style: TextStyle(
                color: Color(0xFF9CA3AF),
                fontSize: 14,
                height: 1.5,
              ),
              textAlign: TextAlign.center,
            ),
            if (_error != null) ...[
              const SizedBox(height: 16),
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: const Color(0x1FFE4A49),
                  borderRadius: BorderRadius.circular(10),
                  border: Border.all(color: const Color(0x66FE4A49)),
                ),
                child: Text(
                  _error!,
                  style: const TextStyle(color: Color(0xFFFE4A49), fontSize: 13),
                  textAlign: TextAlign.center,
                ),
              ),
            ],
            const SizedBox(height: 32),
            _buildPlanCard(
              context: context,
              name: 'Free',
              price: '\$0',
              period: 'forever',
              desc: 'Try Noteika — encrypted notes with semantic search.',
              features: [
                '10 encrypted captures',
                'Semantic + exact search',
                'Project folders',
                'Duplicate warnings',
                'JSON export',
              ],
              ctaText: isPro ? 'Downgrade (via Web)' : 'Current Plan',
              isHighlighted: false,
              isCurrent: !isPro,
              onPressed: null,
            ),
            const SizedBox(height: 24),
            _buildPlanCard(
              context: context,
              name: 'Pro',
              price: '\$8',
              period: '/ month',
              desc: 'Unlimited saves for daily use across projects.',
              features: [
                'Unlimited encrypted captures',
                'Semantic + exact search',
                'Everything in Free',
                'Priority support',
                'Cancel anytime',
              ],
              ctaText: isPro ? 'Current Plan' : 'Upgrade to Pro',
              isHighlighted: true,
              isCurrent: isPro,
              onPressed: isPro || _loading || isIapPending ? null : _upgrade,
            ),
            const SizedBox(height: 32),
          ],
        ),
      ),
    );
  }

  Widget _buildPlanCard({
    required BuildContext context,
    required String name,
    required String price,
    required String period,
    required String desc,
    required List<String> features,
    required String ctaText,
    required bool isHighlighted,
    required bool isCurrent,
    required VoidCallback? onPressed,
  }) {
    final state = context.read<AppState>();
    final showLoader = _loading || (!kIsWeb && state.iapService.purchasePending);

    final borderGradient = isHighlighted
        ? const LinearGradient(
            colors: [Color(0xFF8B5CF6), Color(0xFF6366F1)],
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
          )
        : null;

    final cardContent = Container(
      padding: const EdgeInsets.all(24),
      decoration: BoxDecoration(
        color: const Color(0xFF13151A),
        borderRadius: BorderRadius.circular(15),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text(
                name,
                style: const TextStyle(
                  color: Colors.white,
                  fontWeight: FontWeight.bold,
                  fontSize: 22,
                  letterSpacing: -0.5,
                ),
              ),
              if (isHighlighted)
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                  decoration: BoxDecoration(
                    color: const Color(0x1F8B5CF6),
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(color: const Color(0x668B5CF6)),
                  ),
                  child: const Text(
                    'Unlimited saves',
                    style: TextStyle(
                      color: Color(0xFFA78BFA),
                      fontSize: 11,
                      fontWeight: FontWeight.w600,
                      letterSpacing: 0.5,
                    ),
                  ),
                ),
            ],
          ),
          const SizedBox(height: 16),
          Row(
            crossAxisAlignment: CrossAxisAlignment.baseline,
            textBaseline: TextBaseline.alphabetic,
            children: [
              Text(
                price,
                style: const TextStyle(
                  color: Colors.white,
                  fontWeight: FontWeight.w800,
                  fontSize: 40,
                  letterSpacing: -1,
                ),
              ),
              const SizedBox(width: 6),
              Text(
                period,
                style: const TextStyle(
                  color: Color(0xFF9CA3AF),
                  fontSize: 14,
                  fontWeight: FontWeight.w500,
                ),
              ),
            ],
          ),
          const SizedBox(height: 10),
          Text(
            desc,
            style: const TextStyle(
              color: Color(0xFF9CA3AF),
              fontSize: 14,
              height: 1.45,
            ),
          ),
          const SizedBox(height: 24),
          const Divider(color: Color(0xFF1F2228), height: 1),
          const SizedBox(height: 20),
          ...features.map((feat) => Padding(
                padding: const EdgeInsets.only(bottom: 12),
                child: Row(
                  children: [
                    const Icon(Icons.check_circle_rounded, color: Color(0xFF10B981), size: 18),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Text(
                        feat,
                        style: const TextStyle(
                          color: Color(0xFFD1D5DB),
                          fontSize: 14,
                          fontWeight: FontWeight.w400,
                        ),
                      ),
                    ),
                  ],
                ),
              )),
          const SizedBox(height: 20),
          if (isCurrent)
            OutlinedButton(
              onPressed: null,
              style: OutlinedButton.styleFrom(
                side: const BorderSide(color: Color(0xFF1F2228)),
                disabledForegroundColor: const Color(0xFF6B7280),
                padding: const EdgeInsets.symmetric(vertical: 14),
                shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
              ),
              child: Text(ctaText),
            )
          else if (onPressed == null)
            ElevatedButton(
              onPressed: null,
              style: ElevatedButton.styleFrom(
                backgroundColor: const Color(0xFF1F2228),
                disabledForegroundColor: const Color(0xFF6B7280),
                padding: const EdgeInsets.symmetric(vertical: 14),
                shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
              ),
              child: Text(ctaText),
            )
          else
            ElevatedButton(
              onPressed: onPressed,
              style: ElevatedButton.styleFrom(
                backgroundColor: const Color(0xFF8B5CF6),
                foregroundColor: Colors.white,
                padding: const EdgeInsets.symmetric(vertical: 14),
                shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
              ),
              child: showLoader
                  ? const SizedBox(
                      height: 18,
                      width: 18,
                      child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                    )
                  : Text(ctaText),
            ),
        ],
      ),
    );

    return Container(
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(16),
        gradient: borderGradient,
        border: isHighlighted
            ? null
            : Border.all(color: const Color(0xFF1F2228), width: 1),
        boxShadow: isHighlighted
            ? [
                BoxShadow(
                  color: const Color(0xFF8B5CF6).withOpacity(0.12),
                  blurRadius: 24,
                  spreadRadius: 2,
                  offset: const Offset(0, 4),
                )
              ]
            : null,
      ),
      padding: isHighlighted ? const EdgeInsets.all(1.5) : EdgeInsets.zero,
      child: cardContent,
    );
  }
}

