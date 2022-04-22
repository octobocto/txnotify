import './App.scss';
import { useEffect, useState } from 'react';
import {
  Notification,
  useCreateNotification,
  useCreateUser,
  useListNotifications,
} from './api/txnotify';
import {
  Button,
  TextField,
  createMuiTheme,
  ThemeProvider,
  Card,
  CardContent,
  Typography,
} from '@material-ui/core';
import { BrowserRouter as Router, Switch, Route, Link } from 'react-router-dom';
import { ToastContainer, toast } from 'react-toastify';
import 'react-toastify/dist/ReactToastify.css';
import { useMount } from 'react-use';

const App = () => {
  const { mutate: createUser, error: createUserError } = useCreateUser({});
  const [userID, setUserID] = useState(localStorage.getItem('userID'));

  useMount(() => {
    if (userID === null || userID === '') {
      // we only want to create a new user if one doesn't exist yet
      createUser().then((user) => {
        if (!user.id) {
          console.error('got undefined user ID');
          return;
        }
        setUserID(user.id);
        localStorage.setItem('userID', user.id);
      });
    }
  });

  const recreateUser = () => {
    localStorage.removeItem('userID');
    return createUser().then((user) => {
      if (user.id) {
        setUserID(user.id);
        localStorage.setItem('userID', user.id);
        toast.success(
          'Your old user no longer exists. Successfully created new one.'
        );
        return;
      }
      toast.error(
        'Your old user no longer exists, and could not create new one.'
      );
    });
  };

  // post error notification every time an error occurs
  useEffect(() => {
    const error = createUserError as any;
    const notifyError = () =>
      toast.error(error?.data?.message || createUserError?.message);
    if (createUserError) {
      notifyError();
    }
  }, [createUserError]);

  return (
    <ThemeProvider theme={theme}>
      <Router>
        <ToastContainer />
        <div className="app">
          <div className="nav">
            <div className="link">
              <Link to="/">Home</Link>
            </div>
            <div className="link">
              <Link to="/my-notifications">My notifications</Link>
            </div>
            <div className="link">
              <Link to="/about">About</Link>
            </div>
          </div>

          <div>
            <h1>TX Notify</h1>
            <Switch>
              <Route path="/about">
                <About />
              </Route>
              <Route path="/my-notifications">
                <MyNotifications userID={userID === null ? '' : userID} />
              </Route>
              <Route path="/">
                <Home
                  recreateUser={recreateUser}
                  userID={userID === null ? '' : userID}
                />
              </Route>
            </Switch>
          </div>
        </div>
      </Router>
    </ThemeProvider>
  );
};

export default App;

const About = () => {
  return (
    <div className="about">
      <h2>About</h2>
      <a href="https://docs.txnotify.com">Read our docs</a>
      <h3>What can you do?</h3>
      <p>
        With this website you can get email notifications every time your
        bitcoin address receives money. You can specify the number of
        confirmations yourself, even 0-conf. Every time the specified address
        receives a transaction, tx notify will send you an email. Either the
        second the transaction is broadcasted, or when it reaches your specified
        number of transactions.
      </p>
      <p>
        With this website, you no longer need to refresh a block explorer every
        minute waiting for your transaction to be confirmed. Just pop your
        bitcoin address or txid in here, and tx notify will make sure to let you
        know when the bitcoin transaction is ready. No registration required.
        Even better, it's free!
      </p>
      <h3>
        Reach out to support@txnotify.com if there's any other features you want
      </h3>
      <p>
        The site is under active development. If there's any feature you want,
        pop us an email. No request is too large. Whatever it is, reach out and
        we'll figure something out. We're very flexible.
      </p>
      <p>
        Maybe you want other information about the transaction in the email? Or
        you want notifications somewhere else, for example slack. Or you want to
        integrate tx notify into your own system programatically using our API.
      </p>
    </div>
  );
};

interface Props {
  userID: string;
}

const MyNotifications = (props: Props) => {
  const { data, error } = useListNotifications({
    queryParams: { user_id: props.userID },
  });

  if (error) {
    return <div>Could not find your notifications: {error}</div>;
  }

  return (
    <div className="my-notifications">
      Here's all your current notifications:
      {data?.notifications?.map((v) => {
        return (
          <ul>
            <Notifications notification={v} />
          </ul>
        );
      })}
    </div>
  );
};

const Notifications = ({ notification }: { notification: Notification }) => {
  return (
    <Card>
      <CardContent>
        <Typography color="textSecondary">
          {notification.identifier && notification.identifier.length > 60
            ? 'TXID'
            : 'Bitcoin Address'}
        </Typography>
        <Typography variant="body2" component="p">
          {notification.identifier}
        </Typography>

        <Typography color="textSecondary">Email</Typography>
        <Typography variant="body2" component="p">
          {notification.email}
        </Typography>

        <Typography color="textSecondary">Confirmations</Typography>
        <Typography variant="body2" component="p">
          {notification.confirmations}
        </Typography>

        <Typography color="textSecondary">Description</Typography>
        <Typography variant="body2" component="p">
          {notification.description}
        </Typography>
      </CardContent>
    </Card>
  );
};

interface HomeProps {
  userID: string;

  recreateUser: () => void;
}

const Home: React.FC<HomeProps> = ({ userID, recreateUser }) => {
  const [email, setEmail] = useState('');
  const [description, setDescription] = useState('');
  const [identifier, setId] = useState('');
  const [confirmations, setConfirmations] = useState<number | undefined>();

  const [type, setType] = useState<'address' | 'txid'>(); // txid or address
  const setIdentifier = (v: string) => {
    setId(v);
    if (v.length > 50) {
      setType('txid');
    } else {
      setType('address');
    }
  };

  const {
    error: createNotificationError,
    mutate: createNotification,
  } = useCreateNotification({});

  const notifySuccess = () =>
    toast.success(`Successfully subscribed to notifications for ${type}`);

  // post error notification every time an error occurs
  // retry if the user got a notifications_user_id_fkey error
  useEffect(() => {
    const error = createNotificationError as any;
    const notifyError = () => {
      toast.error(error?.data?.message || createNotificationError?.message);
      if (
        error?.data?.message.includes('notifications_user_id_fkey') ||
        error?.message.includes('notifications_user_id_fkey')
      ) {
        recreateUser();
      }
    };
    if (createNotificationError) {
      notifyError();
    }
  }, [createNotificationError]);

  const apiURL = window.runtimeEnv.API_URL;
  if (!apiURL || apiURL === '') {
    throw Error('api url is not defined');
  }

  return (
    <div className="home">
      <div className="field">
        <TextField
          label="Email"
          type="text"
          value={email}
          onChange={(e) => {
            setEmail(e.currentTarget.value);
          }}
        />
      </div>
      <div className="field">
        <TextField
          label="Bitcoin address or txid"
          type="text"
          value={identifier}
          onChange={(e) => {
            setIdentifier(e.currentTarget.value);
          }}
        />
      </div>
      <div className="field">
        <TextField
          label="Confirmations"
          helperText={
            type === 'txid' && confirmations === 0
              ? 'Confirmations must be greater than 0 if you want to get notified about a transaction.'
              : 'If not set, sends an email at 0 confirmations'
          }
          type="number"
          value={confirmations === undefined ? '' : confirmations}
          error={type === 'txid' && confirmations === 0}
          onChange={(e) => {
            setConfirmations(
              e.currentTarget.value === undefined
                ? undefined
                : parseInt(e.currentTarget.value)
            );
          }}
        />
      </div>
      <div className="field">
        <TextField
          label="Description"
          helperText="Included in the email"
          type="text"
          value={description}
          onChange={(e) => {
            setDescription(e.currentTarget.value);
          }}
        />
      </div>
      <div className="field">
        <Button
          variant="contained"
          type="button"
          disabled={email === '' || identifier === ''}
          onClick={() => {
            createNotification({
              user_id: userID,
              identifier,
              email,
              confirmations: confirmations ? confirmations : 0,
              description,
            }).then(() => {
              notifySuccess();
            });
          }}
        >
          Register
        </Button>
      </div>
    </div>
  );
};

declare global {
  interface Window {
    runtimeEnv: {
      API_URL: string;
    };
  }
}

const theme = createMuiTheme({
  typography: {
    fontFamily: ['Lato', 'Roboto', 'sans-serif'].join(','),
    fontSize: 16,
  },
});
