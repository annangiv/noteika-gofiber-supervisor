import 'dart:async';
import 'dart:io';
import 'package:flutter/foundation.dart';
import 'package:in_app_purchase/in_app_purchase.dart';
import '../config/api_config.dart';
import '../state/app_state.dart';

class IapService {
  IapService(this._appState);

  final AppState _appState;
  final InAppPurchase _iap = InAppPurchase.instance;
  StreamSubscription<List<PurchaseDetails>>? _subscription;

  bool _isAvailable = false;
  List<ProductDetails> _products = [];
  bool _purchasePending = false;
  String? _errorMessage;

  bool get isAvailable => _isAvailable;
  List<ProductDetails> get products => _products;
  bool get purchasePending => _purchasePending;
  String? get errorMessage => _errorMessage;

  void initialize() {
    final purchaseUpdated = _iap.purchaseStream;
    _subscription = purchaseUpdated.listen(
      _onPurchaseUpdate,
      onError: (error) {
        debugPrint('[IAP] Purchase stream error: $error');
        _errorMessage = error.toString();
        _purchasePending = false;
        _appState.notify();
      },
    );
    _checkAvailabilityAndLoadProducts();
  }

  void dispose() {
    _subscription?.cancel();
  }

  Future<void> _checkAvailabilityAndLoadProducts() async {
    try {
      _isAvailable = await _iap.isAvailable();
      if (!_isAvailable) {
        debugPrint('[IAP] In-app purchase not available on this device');
        return;
      }

      // Query products
      final response = await _iap.queryProductDetails({ApiConfig.proSubscriptionId});
      if (response.notFoundIDs.isNotEmpty) {
        debugPrint('[IAP] Products not found: ${response.notFoundIDs}');
      }
      _products = response.productDetails;
      _appState.notify();
    } catch (e) {
      debugPrint('[IAP] Error loading products: $e');
    }
  }

  Future<void> buyProSubscription() async {
    if (!_isAvailable || _products.isEmpty) {
      _errorMessage = 'Store is currently unavailable. Please try again later.';
      _appState.notify();
      return;
    }

    final product = _products.firstWhere(
      (p) => p.id == ApiConfig.proSubscriptionId,
      orElse: () => throw Exception('Pro Subscription product not found'),
    );

    _purchasePending = true;
    _errorMessage = null;
    _appState.notify();

    try {
      final purchaseParam = PurchaseParam(productDetails: product);
      await _iap.buyNonConsumable(purchaseParam: purchaseParam);
    } catch (e) {
      _purchasePending = false;
      _errorMessage = e.toString();
      _appState.notify();
    }
  }

  Future<void> _onPurchaseUpdate(List<PurchaseDetails> purchases) async {
    for (final purchase in purchases) {
      if (purchase.status == PurchaseStatus.pending) {
        _purchasePending = true;
        _appState.notify();
      } else if (purchase.status == PurchaseStatus.error) {
        _purchasePending = false;
        _errorMessage = purchase.error?.message ?? 'Purchase failed';
        _appState.notify();
        if (purchase.pendingCompletePurchase) {
          await _iap.completePurchase(purchase);
        }
      } else if (purchase.status == PurchaseStatus.purchased ||
                 purchase.status == PurchaseStatus.restored) {
        final platform = Platform.isAndroid ? 'android' : 'ios';
        final token = purchase.verificationData.serverVerificationData;
        
        debugPrint('[IAP] Purchase successful. Verifying token on backend: $token');

        try {
          final res = await _appState.api.verifyIapPurchase(
            platform: platform,
            purchaseToken: token,
            productId: purchase.productID,
          );
          
          if (res['success'] == true) {
            debugPrint('[IAP] Purchase verified successfully. Syncing user state...');
            await _appState.bootstrap();
          }
        } catch (e) {
          debugPrint('[IAP] Failed to verify purchase on backend: $e');
          _errorMessage = 'Purchase completed but failed to verify on server: $e';
        } finally {
          _purchasePending = false;
          _appState.notify();
          if (purchase.pendingCompletePurchase) {
            await _iap.completePurchase(purchase);
          }
        }
      } else if (purchase.status == PurchaseStatus.canceled) {
        _purchasePending = false;
        _appState.notify();
      }
    }
  }
}
