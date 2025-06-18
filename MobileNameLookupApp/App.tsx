import React, { useState } from 'react';
import { StatusBar } from 'expo-status-bar';
import { StyleSheet, Text, View, TextInput, TouchableOpacity, ActivityIndicator, KeyboardAvoidingView, Platform } from 'react-native';
import axios from 'axios';

const API_URL = 'https://spiteful-seashore-production.up.railway.app/lookup'; // <-- Replace with your deployed Railway API endpoint

export default function App() {
  const [mobile, setMobile] = useState('');
  const [result, setResult] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const handleLookup = async () => {
    setResult(null);
    setError(null);
    setLoading(true);
    try {
      const form = new FormData();  
      form.append('mobile', mobile);
      const response = await axios.post(API_URL, form, {
        headers: { 'Content-Type': 'multipart/form-data' },
      });
      if (response.data && response.data.result && response.data.result.mobile_linked_name) {
        setResult(response.data.result.mobile_linked_name);
      } else if (response.data && response.data.message) {
        setError(response.data.message);
      } else {
        setError('No name found for this number.');
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'Lookup failed. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <KeyboardAvoidingView style={{ flex: 1 }} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
      <View style={styles.container}>
        <Text style={styles.title}>Mobile Name Lookup</Text>
        <TextInput
          style={styles.input}
          placeholder="Enter mobile number (e.g. +91 83180 90007)"
          keyboardType="phone-pad"
          value={mobile}
          onChangeText={setMobile}
          autoCapitalize="none"
        />
        <TouchableOpacity style={styles.button} onPress={handleLookup} disabled={loading}>
          <Text style={styles.buttonText}>{loading ? 'Looking up...' : 'Lookup'}</Text>
        </TouchableOpacity>
        {loading && <ActivityIndicator style={{ marginTop: 20 }} />}
        {result && (
          <View style={styles.resultBox}>
            <Text style={styles.resultLabel}>Name:</Text>
            <Text style={styles.resultText}>{result}</Text>
          </View>
        )}
        {error && <Text style={styles.error}>{error}</Text>}
        <StatusBar style="auto" />
      </View>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f5f5f5',
    alignItems: 'center',
    justifyContent: 'center',
    padding: 20,
  },
  title: {
    fontSize: 28,
    fontWeight: 'bold',
    marginBottom: 30,
    color: '#333',
  },
  input: {
    width: '100%',
    padding: 15,
    borderColor: '#ccc',
    borderWidth: 1,
    borderRadius: 8,
    marginBottom: 20,
    backgroundColor: '#fff',
    fontSize: 18,
  },
  button: {
    width: '100%',
    backgroundColor: '#4CAF50',
    padding: 15,
    borderRadius: 8,
    alignItems: 'center',
    marginBottom: 20,
  },
  buttonText: {
    color: '#fff',
    fontSize: 18,
    fontWeight: 'bold',
  },
  resultBox: {
    marginTop: 30,
    backgroundColor: '#e9ecef',
    padding: 20,
    borderRadius: 8,
    alignItems: 'center',
  },
  resultLabel: {
    fontSize: 16,
    color: '#495057',
    marginBottom: 5,
  },
  resultText: {
    fontSize: 22,
    fontWeight: 'bold',
    color: '#333',
  },
  error: {
    color: '#dc3545',
    marginTop: 20,
    fontSize: 16,
    textAlign: 'center',
  },
});
