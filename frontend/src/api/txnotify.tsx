/* eslint-disable */
/* Generated by restful-react */

import React from "react";
import { Get, GetProps, useGet, UseGetProps, Mutate, MutateProps, useMutate, UseMutateProps } from "restful-react";
export interface CreateNotificationResponse {
  /**
   * the id of your notification. Can be used to get more specific information about your subscription,
   * or to delete it.
   */
  id?: string;
}

export interface CreateUserResponse {
  id?: string;
}

export interface ListNotificationsResponse {
  notifications?: Notification[];
}

export interface Notification {
  user_id?: string;
  /**
   * The bitcoin blockchain id of the transaction you want to monitor or
   * the bitcoin blockchain address you want to monitor. You will be notified about all
   * new transactions sent to and from this address.
   */
  identifier?: string;
  /**
   * how many confirmations the transaction should have when you want to be notified. Can not be
   * higher than 6. If omitted, you will get a notification at 0 confirmations.
   */
  confirmations?: number;
  email?: string;
  description?: string;
  slack_webhook_url?: string;
  callback_url?: string;
}

export interface ListNotificationsQueryParams {
  user_id?: string;
}

export type ListNotificationsProps = Omit<GetProps<ListNotificationsResponse, unknown, ListNotificationsQueryParams, void>, "path">;

/**
 * ListNotifications can be used to list all your current active notifications
 */
export const ListNotifications = (props: ListNotificationsProps) => (
  <Get<ListNotificationsResponse, unknown, ListNotificationsQueryParams, void>
    path={`/notifications`}
    
    {...props}
  />
);

export type UseListNotificationsProps = Omit<UseGetProps<ListNotificationsResponse, unknown, ListNotificationsQueryParams, void>, "path">;

/**
 * ListNotifications can be used to list all your current active notifications
 */
export const useListNotifications = (props: UseListNotificationsProps) => useGet<ListNotificationsResponse, unknown, ListNotificationsQueryParams, void>(`/notifications`, props);


export type CreateNotificationProps = Omit<MutateProps<CreateNotificationResponse, unknown, void, Notification, void>, "path" | "verb">;

export const CreateNotification = (props: CreateNotificationProps) => (
  <Mutate<CreateNotificationResponse, unknown, void, Notification, void>
    verb="POST"
    path={`/notifications`}
    
    {...props}
  />
);

export type UseCreateNotificationProps = Omit<UseMutateProps<CreateNotificationResponse, unknown, void, Notification, void>, "path" | "verb">;

export const useCreateNotification = (props: UseCreateNotificationProps) => useMutate<CreateNotificationResponse, unknown, void, Notification, void>("POST", `/notifications`, props);


export type CreateUserProps = Omit<MutateProps<CreateUserResponse, unknown, void, void, void>, "path" | "verb">;

export const CreateUser = (props: CreateUserProps) => (
  <Mutate<CreateUserResponse, unknown, void, void, void>
    verb="POST"
    path={`/users`}
    
    {...props}
  />
);

export type UseCreateUserProps = Omit<UseMutateProps<CreateUserResponse, unknown, void, void, void>, "path" | "verb">;

export const useCreateUser = (props: UseCreateUserProps) => useMutate<CreateUserResponse, unknown, void, void, void>("POST", `/users`, props);
