// Copyright (c) 2021 Chair of Applied Cryptography, Technische Universität
// Darmstadt, Germany. All rights reserved. This file is part of
// perun-eth-mobile. Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package network.perun.app;

import android.app.Activity;
import android.os.Bundle;
import android.util.Log;
import android.widget.TextView;

import java.util.Arrays;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.nio.ByteBuffer;
import java.math.BigInteger;

import prnm.*;

public class MainActivity extends Activity {
    Node node;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        try {
            // Setting the log level to Trace, default is Info.
            Prnm.setLogLevel(6);
            // Waiting for the Android Debugger to attach the logger.
            Thread.sleep(2000);
            // Get the Apps data directory.
            String appDir = getApplicationContext().getFilesDir().getAbsolutePath();
            String ksPath = appDir +"/keystore";
            String dbPath = appDir +"/database";
            // Set a wallet password. Do not use prnm with real funds right now!
            String password = "0123456789";
            // Your onChain Ethereum secret-key
            String sk = "0x69cb97043e56883d66627e8f7a828877a56022d0fb05ae6197e6e16fb56282d0";

            // Create a wallet.
            Wallet wallet = Prnm.newWallet(ksPath, password);
            // Import the secret key.
            Address onChain = wallet.importAccount(sk);
            Log.i("prnm", "Address: " +onChain.toHex());
            // 10.0.2.2 is the IP of the host PC when using Android Simulator and the host is running a ganache-cli.
            // 8545 is the standard port of ganache-cli.
            String ethUrl = "ws://10.0.2.2:8545";

            // Using null as either Adjudicator or AssetHolder tells the Client to deploy the contracts,
            // in this case we already deployed them and enter their addresses.
            Address adjudicator = new Address("0xDc4A7e107aD6dBDA1870df34d70B51796BBd1335");
            Address assetHolder = new Address("0xb051EAD0C6CC2f568166F8fEC4f07511B88678bA");
            // Define how many blocks a transaction needs to be part of to be considered final.
            int txFinalityDepth = 1;
            // We will be listening on 127.0.0.1:5750 for new channel proposals with the alias "Alice".
            Config cfg = new Config("Alice", onChain, adjudicator, assetHolder, ethUrl, "127.0.0.1", 5750, txFinalityDepth);
            node = new Node(cfg, wallet);
            Address bob = new Address("0xA298Fc05bccff341f340a11FffA30567a00e651f");
            // Create the initial balances of the channel, we start with 2000 and bob with 1000.
            node.addPeer(bob, "10.0.2.2", 5750);

            // (Optional) Enable the persistence and reconnect to peers:
            //
            // EnablePersistence attempts to load the database from the given path or creates
            // a new one. It then retrieves all channels from the database.
            node.enablePersistence(dbPath);
            // Restore tries to reestablish connections to all previously connected peers.
            // It needs to be called only once, but all peers need to be added beforehand.
            node.restore();

            // (Optional) Propose a channel to bob:
            // (Without the following lines, the node will still accept incoming channel proposals.)
            //
            // PerunId (currently an Address) of the peer that we want to open a channel with.
            BigInts initBals = Prnm.newBalances(new BigInt("2000000000000000000", 10), new BigInt("1000000000000000000", 10));
            // Send the proposal. Once the channel was accepted, the node will handle the
            // channel updates and onChain watching.
            node.propose(bob, initBals);
        } catch (Exception e) {
            Log.e("prnm", e.toString());
        }
    }

    @Override
    protected void onDestroy() {
        super.onDestroy();
        node.close();
    }
}

class Node implements prnm.NewChannelCallback, prnm.ProposalHandler, prnm.UpdateHandler, prnm.ConcludedEventHandler {
    public Client client;
    // Since java uses _pointer comparison_ for byte[] keys in a map, we need to wrap it in ByteBuffer.
    public Map<ByteBuffer, PaymentChannel> chs = new ConcurrentHashMap<ByteBuffer, PaymentChannel>();

    public Node(Config cfg, Wallet wallet) throws Exception {
        // Possibly has to deploy contracts, so give it some extra time.
        Context ctx = Prnm.contextWithTimeout(600);
        try {
            client = new Client(ctx, cfg, wallet);
            // Set the handler for new channels.
            client.onNewChannel(this);
        } finally {
            ctx.cancel();
        }
        // Start a new thread for handling channel proposals and channel updates.
        new Thread(() -> {
            client.handle(this, this);
        }).start();
    }

    public void restore() throws Exception {
        Context ctx = Prnm.contextWithTimeout(20);
        try {
            client.restore(ctx);
        } finally {
            ctx.cancel();
        }
    }

    public void enablePersistence(String dbPath) throws Exception {
        client.enablePersistence(dbPath);
    }

    public void close() {
        try {
            client.close();
        } catch (Exception e) {
            Log.e("prnm", "Error while closing the client: " + e.toString());
        }
    }

    public void addPeer(prnm.Address peer, String ip, int port) {
        // This is safe to call more than once.
        client.addPeer(peer, ip, port);
    }

    public byte[] propose(Address peer, BigInts initBals) throws Exception {
        // Has to send transactions, so give it some extra time.
        Context ctx = Prnm.contextWithTimeout(600);
        try {
            byte[] id = client.proposeChannel(ctx, peer, 60, initBals).getParams().getID();
            // Retrieve the channel from chs which was inserted by accept.
            PaymentChannel ch = chs.get(ByteBuffer.wrap(id));
            if (ch == null)
                throw new Exception(String.format("propose: channel not found id=%s", new BigInteger(1, id).toString(16)));
            Log.i("prnm", "Proposal to peer " + peer.toHex() + " successful, id: " + ch.getParams().getID());
            return id;
        } finally {
            ctx.cancel();
        }
    }

    // Handles all channel proposals by accepting them.
    @Override
    public void handleProposal(ChannelProposal proposal, ProposalResponder responder) {
        Context ctx = Prnm.contextWithTimeout(600);
        try {
            BigInts bals = proposal.getInitBals();
            Log.i("prnm", String.format("Channel proposal (id=%s, bals=[%d,%d])", proposal.getPeer().toHex(), bals.get(0).toInt64(), bals.get(1).toInt64()));
            byte[] id = responder.accept(ctx).getParams().getID();
            // Retrieve the channel from chs which was inserted by accept.
            PaymentChannel ch = chs.get(ByteBuffer.wrap(id));
            if (ch == null)
                throw new Exception(String.format("accept: channel not found id=%s", new BigInteger(1, id).toString(16)));
             Log.i("prnm", "Accepted new channel proposal (id=" + ch.getParams().getID());
        } catch (Exception e) {
            Log.e("prnm", e.toString());
        } finally {
            ctx.cancel();
        }
    }

    // Handles all new channels by calling `watch` on them.
    @Override
    public void onNew(PaymentChannel channel) {
        byte[] id = channel.getParams().getID();
        if (chs.containsKey(ByteBuffer.wrap(id))) {
            Log.w("prnm", "Overriding Channel " + new BigInteger(1, id).toString(16));
        } else {
            Log.i("prnm", "New channel " + new BigInteger(1, id).toString(16));
        }
        chs.put(ByteBuffer.wrap(id), channel);

        // Start a new thread for watching the channel.
        new Thread(() -> {
            try {
                Log.d("channel", "Starting watching");
                channel.watch(this);
                Log.d("channel", "Stopped watching");
            }  catch (Exception e) {
                Log.e("channel", "Error watching:" + e.toString());
            }
        }).start();
    }

    // Handles all channel updates by accepting them.
    @Override
    public void handleUpdate(ChannelUpdate update, UpdateResponder responder) {
        Context ctx = Prnm.contextWithTimeout(5);
        try {
            State state = update.getState();
            Log.i("channel", String.format("Update (version=%d, isFinal=%b)", state.getVersion(), state.isFinal()));
            responder.accept(ctx);
            BigInts bals = update.getState().getBalances();
            Log.d("channel", String.format("Accepted update (version=%d, bals=[%s, %s])", state.getVersion(), bals.get(update.getActorIdx()).toString(), bals.get(1-update.getActorIdx()).toString()));
        } catch (Exception e) {
            Log.e("channel", e.toString());
        } finally {
            ctx.cancel();
        }
    }

    // Handles all channel conclusion events on the Adjudicator.
    @Override
    public void handleConcluded(byte[] id) {
        Log.i("channel", "Received concluded event for channel " + new BigInteger(1, id).toString(16));
        Context ctx = Prnm.contextWithTimeout(30);
        try {
            PaymentChannel ch = chs.get(ByteBuffer.wrap(id));
            if (ch == null) {
                // If we initiated the channel closing, then the channel should
                // already be removed and we return.
                return;
            }
            ch.settle(ctx, true);
            chs.remove(ByteBuffer.wrap(id));
            Log.i("channel", "Settled channel " + new BigInteger(1, id).toString(16));
        } catch (Exception e) {
            Log.e("channel", e.toString());
        } finally {
            ctx.cancel();
        }
    }
}
